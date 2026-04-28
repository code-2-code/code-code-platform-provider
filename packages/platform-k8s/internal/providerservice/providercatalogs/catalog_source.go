package providercatalogs

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	"code-code.internal/platform-k8s/internal/modelservice/modelcatalogdiscovery"
)

// CatalogProbeRequest describes one model catalog probe invocation.
type CatalogProbeRequest struct {
	ProbeID                  string
	Protocol                 apiprotocolv1.Protocol
	BaseURL                  string
	Headers                  http.Header
	ProviderSurfaceBindingID string
	Operation                *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation
	DynamicValues            modelcatalogdiscovery.DynamicValues
	ConcurrencyKey           string
}

// CatalogModelIDProbe probes a provider endpoint for discoverable model IDs.
type CatalogModelIDProbe interface {
	ProbeModelIDs(context.Context, CatalogProbeRequest) ([]string, error)
}

// catalogSourceRef identifies one registered catalog source.
type catalogSourceRef struct {
	ID string
}

func newCatalogSourceRef(probeID string) catalogSourceRef {
	return catalogSourceRef{ID: strings.TrimSpace(probeID)}
}

func (r catalogSourceRef) key() (string, error) {
	id := strings.TrimSpace(r.ID)
	if id == "" {
		return "", fmt.Errorf("platformk8s/providercatalogs: probe id is empty")
	}
	return id, nil
}

// catalogSource describes a model catalog capability that can be registered.
type catalogSource interface {
	ref() catalogSourceRef
}

// catalogSourceRegistry tracks registered catalog sources.
type catalogSourceRegistry struct {
	sources map[string]catalogSource
}

func newCatalogSourceRegistry() *catalogSourceRegistry {
	return &catalogSourceRegistry{sources: map[string]catalogSource{}}
}

func (r *catalogSourceRegistry) register(source catalogSource) error {
	if r == nil {
		return fmt.Errorf("platformk8s/providercatalogs: registry is nil")
	}
	if source == nil {
		return fmt.Errorf("platformk8s/providercatalogs: source is nil")
	}
	key, err := source.ref().key()
	if err != nil {
		return err
	}
	if _, exists := r.sources[key]; exists {
		return fmt.Errorf("platformk8s/providercatalogs: source %q is already registered", key)
	}
	r.sources[key] = source
	return nil
}

func (r *catalogSourceRegistry) has(ref catalogSourceRef) bool {
	if r == nil {
		return false
	}
	key, err := ref.key()
	if err != nil {
		return false
	}
	return r.sources[key] != nil
}
