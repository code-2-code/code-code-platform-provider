package providercatalogs

import (
	"context"
	"net/http"
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	"code-code.internal/platform-k8s/internal/platform/modelcatalogdiscovery"
)

type CatalogModelIDExecutor interface {
	ProbeModelIDs(ctx context.Context, request CatalogProbeRequest) ([]string, error)
}

type CatalogProbeHeaderResolver interface {
	ResolveCatalogProbeHeaders(ctx context.Context, request CatalogProbeHeaderRequest) (http.Header, error)
}

type CatalogProbeHeaderRequest struct {
	CredentialID string
	Protocol     apiprotocolv1.Protocol
	BaseURL      string
	Operation    *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation
}

// materializerProbe adapts CatalogProbeExecutor to the ModelIDProbe interface
// used by the materializer. It translates the simplified ProbeRequest into a
// full CatalogProbeRequest.
type materializerProbe struct {
	executor       CatalogModelIDExecutor
	headerResolver CatalogProbeHeaderResolver
}

// NewMaterializerProbe creates a ModelIDProbe backed by a CatalogProbeExecutor.
func NewMaterializerProbe(executor CatalogModelIDExecutor, headerResolver CatalogProbeHeaderResolver) ModelIDProbe {
	return &materializerProbe{executor: executor, headerResolver: headerResolver}
}

func (p *materializerProbe) ProbeModelIDs(ctx context.Context, request ProbeRequest) ([]string, error) {
	protocol := request.Protocol
	operation := request.Operation
	if operation == nil {
		operation = modelcatalogdiscovery.DefaultAPIKeyDiscoveryOperation(protocol)
	}
	headers, err := p.catalogProbeHeaders(ctx, CatalogProbeHeaderRequest{
		CredentialID: strings.TrimSpace(request.ProviderCredentialID),
		Protocol:     protocol,
		BaseURL:      strings.TrimSpace(request.BaseURL),
		Operation:    operation,
	})
	if err != nil {
		return nil, err
	}
	internal := CatalogProbeRequest{
		ProbeID:        request.ProbeID,
		Protocol:       protocol,
		BaseURL:        strings.TrimSpace(request.BaseURL),
		Headers:        headers,
		SurfaceID:      strings.TrimSpace(request.SurfaceID),
		Operation:      operation,
		ConcurrencyKey: strings.TrimSpace(request.TargetID),
	}
	return p.executor.ProbeModelIDs(ctx, internal)
}

func (p *materializerProbe) catalogProbeHeaders(ctx context.Context, request CatalogProbeHeaderRequest) (http.Header, error) {
	if !operationRequiresSecurity(request.Operation) {
		return nil, nil
	}
	if p.headerResolver == nil {
		return nil, modelCatalogProbeAuthError("auth header resolver is not configured")
	}
	return p.headerResolver.ResolveCatalogProbeHeaders(ctx, request)
}
