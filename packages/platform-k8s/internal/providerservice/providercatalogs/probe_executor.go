package providercatalogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code-code.internal/go-contract/domainerror"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	"code-code.internal/platform-k8s/internal/modelservice/modelcatalogdiscovery"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/encoding/protojson"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CatalogProbeExecutor struct {
	httpClientFactory modelcatalogdiscovery.HTTPClientFactory
	client            ctrlclient.Client
	leaseNamespace    string
	group             singleflight.Group
	metrics           *catalogProbeMetrics
}

func NewCatalogProbeExecutor(
	httpClientFactory modelcatalogdiscovery.HTTPClientFactory,
	client ctrlclient.Client,
	leaseNamespace string,
) (*CatalogProbeExecutor, error) {
	if httpClientFactory == nil {
		return nil, fmt.Errorf("platformk8s/providercatalogs: http client factory is nil")
	}
	if client == nil {
		return nil, fmt.Errorf("platformk8s/providercatalogs: client is nil")
	}
	if strings.TrimSpace(leaseNamespace) == "" {
		return nil, fmt.Errorf("platformk8s/providercatalogs: lease namespace is empty")
	}
	metrics, err := registerCatalogProbeMetrics()
	if err != nil {
		return nil, err
	}
	return &CatalogProbeExecutor{
		httpClientFactory: httpClientFactory,
		client:            client,
		leaseNamespace:    strings.TrimSpace(leaseNamespace),
		metrics:           metrics,
	}, nil
}

func (e *CatalogProbeExecutor) ProbeModelIDs(ctx context.Context, request CatalogProbeRequest) ([]string, error) {
	if e == nil {
		return nil, fmt.Errorf("platformk8s/providercatalogs: catalog probe executor is nil")
	}
	operation := request.Operation
	if operation == nil {
		operation = modelcatalogdiscovery.DefaultAPIKeyDiscoveryOperation(request.Protocol)
	}
	requestKey := catalogProbeSingleflightKey(request, operation)
	leaseKey := catalogProbeLeaseKey(request, requestKey)
	value, err, _ := e.group.Do(requestKey, func() (any, error) {
		started := time.Now()
		var modelIDs []string
		var probeErr error
		defer func() {
			e.metrics.record(request, operation, len(modelIDs), started, probeErr)
		}()
		release, lockErr := e.acquireCatalogProbeLease(ctx, leaseKey)
		if lockErr != nil {
			probeErr = lockErr
			return nil, lockErr
		}
		defer release()
		modelIDs, probeErr = e.probe(ctx, request, operation)
		return modelIDs, probeErr
	})
	if err != nil {
		return nil, err
	}
	modelIDs, _ := value.([]string)
	return modelIDs, nil
}

func (e *CatalogProbeExecutor) probe(
	ctx context.Context,
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
) ([]string, error) {
	var body []byte
	var err error
	if operationUsesCredential(operation, request) {
		body, err = e.fetchAuthenticated(ctx, request, operation)
	} else {
		body, err = e.fetchAnonymous(ctx, request, operation)
	}
	if err != nil {
		return nil, err
	}
	modelIDs, err := modelcatalogdiscovery.ParseModelIDs(body, operation.GetResponseKind())
	if err != nil {
		return nil, err
	}
	if len(modelIDs) == 0 {
		return nil, domainerror.NewValidation("platformk8s/providercatalogs: no models discovered from operation")
	}
	return modelIDs, nil
}

func (e *CatalogProbeExecutor) fetchAnonymous(
	ctx context.Context,
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
) ([]byte, error) {
	service, err := modelcatalogdiscovery.NewService(e.httpClientFactory)
	if err != nil {
		return nil, err
	}
	response, err := service.Fetch(ctx, modelcatalogdiscovery.Request{
		BaseURL:       request.BaseURL,
		Headers:       request.Headers,
		Operation:     operation,
		DynamicValues: request.DynamicValues,
	})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func (e *CatalogProbeExecutor) fetchAuthenticated(
	ctx context.Context,
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
) ([]byte, error) {
	providerSurfaceBindingID := strings.TrimSpace(request.ProviderSurfaceBindingID)
	if providerSurfaceBindingID == "" {
		return nil, domainerror.NewValidation("platformk8s/providercatalogs: auth-bound model operation requires provider_surface_binding_id")
	}
	auth, authHeaders, err := modelcatalogdiscovery.EnvoyAuthContextForOperation(
		request.Protocol,
		providerSurfaceBindingID,
		operation,
	)
	if err != nil {
		return nil, err
	}
	headers := request.Headers.Clone()
	for name, values := range authHeaders {
		for _, value := range values {
			headers.Add(name, value)
		}
	}
	service, err := modelcatalogdiscovery.NewService(e.httpClientFactory)
	if err != nil {
		return nil, err
	}
	response, err := service.Fetch(ctx, modelcatalogdiscovery.Request{
		BaseURL:       request.BaseURL,
		Headers:       headers,
		Operation:     operation,
		DynamicValues: request.DynamicValues,
		EnvoyAuth:     auth,
	})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func operationUsesCredential(operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation, request CatalogProbeRequest) bool {
	if len(operation.GetSecurity()) > 0 {
		return true
	}
	return strings.TrimSpace(request.ProviderSurfaceBindingID) != ""
}

func catalogProbeSingleflightKey(
	request CatalogProbeRequest,
	operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation,
) string {
	operationJSON, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(operation)
	if err != nil {
		operationJSON = []byte(operation.String())
	}
	parts := []string{
		request.Protocol.String(),
		strings.TrimSpace(request.BaseURL),
		strings.TrimSpace(request.ProviderSurfaceBindingID),
		string(operationJSON),
	}
	return strings.Join(parts, "\x00")
}

func catalogProbeLeaseKey(request CatalogProbeRequest, fallback string) string {
	key := strings.TrimSpace(request.ConcurrencyKey)
	if key != "" {
		return key
	}
	return fallback
}
