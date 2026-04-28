package protocolruntime

import (
	"context"
	"sync"
	"time"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BaseRuntime provides shared runtime lifecycle logic for provider
// implementations that expose one preconfigured surface model catalog.
type BaseRuntime struct {
	Surface    *providerv1.ProviderSurfaceBinding
	Credential *credentialcontract.ResolvedCredential
	Now        func() time.Time

	mu      sync.RWMutex
	catalog *providerv1.ProviderModelCatalog
}

// HealthCheck reports whether the runtime surface config is valid enough to
// attempt a call. Model catalog discovery is asynchronous and is not a health
// gate.
func (r *BaseRuntime) HealthCheck(ctx context.Context) error {
	if r == nil || r.Surface == nil {
		return nil
	}
	return providerv1.ValidateProviderSurfaceRuntime(r.Surface.GetRuntime())
}

// ListModels returns the configured surface model catalog for the bound
// surface binding.
func (r *BaseRuntime) ListModels(ctx context.Context) (*providerv1.ProviderModelCatalog, error) {
	r.mu.RLock()
	if r.catalog != nil {
		cached := proto.Clone(r.catalog).(*providerv1.ProviderModelCatalog)
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()
	return r.surfaceCatalog(ctx)
}

// surfaceCatalog returns the pre-configured surface catalog.
func (r *BaseRuntime) surfaceCatalog(_ context.Context) (*providerv1.ProviderModelCatalog, error) {
	ec := r.Surface.GetRuntime().GetCatalog()
	if ec == nil {
		ec = &providerv1.ProviderModelCatalog{}
	}
	catalog := proto.Clone(ec).(*providerv1.ProviderModelCatalog)
	if catalog.Source == providerv1.CatalogSource_CATALOG_SOURCE_UNSPECIFIED {
		catalog.Source = providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG
	}
	if catalog.UpdatedAt == nil && r.Now != nil {
		catalog.UpdatedAt = timestamppb.New(r.Now())
	}
	if len(catalog.GetModels()) > 0 {
		if err := providerv1.ValidateProviderModelCatalog(catalog); err != nil {
			return nil, err
		}
	}
	r.mu.Lock()
	r.catalog = proto.Clone(catalog).(*providerv1.ProviderModelCatalog)
	r.mu.Unlock()
	return catalog, nil
}

// Close releases runtime-owned resources.
func (r *BaseRuntime) Close(_ context.Context) error {
	return nil
}
