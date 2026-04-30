package providers

import (
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type ProviderProjection struct {
	value *managementv1.ProviderView
}

func providerProjectionFromProvider(provider *providerv1.Provider, surface *providerv1.ProviderSurface) *ProviderProjection {
	view := &managementv1.ProviderView{
		ProviderId:           strings.TrimSpace(provider.GetProviderId()),
		DisplayName:          strings.TrimSpace(provider.GetDisplayName()),
		ProviderCredentialId: strings.TrimSpace(provider.GetProviderCredentialRef().GetProviderCredentialId()),
		ModelCatalog:         cloneProviderModelCatalog(provider.GetRuntime().GetCatalog()),
		SurfaceId:            strings.TrimSpace(provider.GetSurfaceId()),
		Runtime:              cloneProviderSurfaceRuntime(provider.GetRuntime()),
		ProductInfoId:        productInfoIDFromProvider(provider, surface),
		Status:               statusFromProvider(provider),
	}
	return &ProviderProjection{value: view}
}

func productInfoIDFromProvider(provider *providerv1.Provider, surface *providerv1.ProviderSurface) string {
	if provider == nil {
		return ""
	}
	runtime := provider.GetRuntime()
	if cliID := strings.TrimSpace(providerv1.RuntimeCLIID(runtime)); cliID != "" {
		return cliID
	}
	if vendorID := vendorProductInfoIDFromRuntime(runtime); vendorID != "" {
		return vendorID
	}
	return productInfoIDFromSurface(surface)
}

func vendorProductInfoIDFromRuntime(runtime *providerv1.ProviderSurfaceRuntime) string {
	if runtime == nil {
		return ""
	}
	const vendorPrefix = "vendor."
	rulesetID := strings.TrimSpace(runtime.GetEgressRulesetId())
	if !strings.HasPrefix(rulesetID, vendorPrefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(rulesetID, vendorPrefix))
}

func productInfoIDFromSurface(surface *providerv1.ProviderSurface) string {
	if surface == nil {
		return ""
	}
	if api := surface.GetApi(); api != nil {
		return strings.TrimSpace(api.GetProductInfoId())
	}
	if cli := surface.GetCli(); cli != nil {
		return strings.TrimSpace(cli.GetProductInfoId())
	}
	if web := surface.GetWeb(); web != nil {
		return strings.TrimSpace(web.GetProductInfoId())
	}
	return ""
}

func statusFromProvider(provider *providerv1.Provider) *managementv1.ProviderStatus {
	if provider == nil {
		return &managementv1.ProviderStatus{
			Phase:  providerservicev1.ProviderPhase_PROVIDER_PHASE_INVALID_CONFIG,
			Reason: "provider is nil",
		}
	}
	if err := providerv1.ValidateProvider(provider); err != nil {
		return &managementv1.ProviderStatus{
			Phase:  providerservicev1.ProviderPhase_PROVIDER_PHASE_INVALID_CONFIG,
			Reason: err.Error(),
		}
	}
	return &managementv1.ProviderStatus{
		Phase: providerservicev1.ProviderPhase_PROVIDER_PHASE_READY,
	}
}

func (p *ProviderProjection) Proto() *managementv1.ProviderView {
	if p == nil || p.value == nil {
		return &managementv1.ProviderView{}
	}
	return proto.Clone(p.value).(*managementv1.ProviderView)
}

func (p *ProviderProjection) ID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetProviderId())
}

func (p *ProviderProjection) DisplayName() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetDisplayName())
}

func (p *ProviderProjection) CredentialID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetProviderCredentialId())
}

func (p *ProviderProjection) SurfaceID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetSurfaceId())
}

func (p *ProviderProjection) AuthKind() providerv1.ProviderSurfaceKind {
	if p == nil || p.value == nil || p.value.GetRuntime() == nil {
		return providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_UNSPECIFIED
	}
	return providerv1.RuntimeKind(p.value.GetRuntime())
}

func (p *ProviderProjection) CLIID() string {
	if p == nil || p.value == nil || p.value.GetRuntime() == nil {
		return ""
	}
	return providerv1.RuntimeCLIID(p.value.GetRuntime())
}

func cloneProviderModelCatalog(catalog *providerv1.ProviderModelCatalog) *providerv1.ProviderModelCatalog {
	if catalog == nil {
		return nil
	}
	return proto.Clone(catalog).(*providerv1.ProviderModelCatalog)
}

func cloneProviderSurfaceRuntime(runtime *providerv1.ProviderSurfaceRuntime) *providerv1.ProviderSurfaceRuntime {
	if runtime == nil {
		return nil
	}
	return proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
}
