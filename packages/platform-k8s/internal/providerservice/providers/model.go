package providers

import (
	"slices"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type ProviderProjection struct {
	value *managementv1.ProviderView
}

func providerProjectionFromProvider(provider *providerv1.Provider) *ProviderProjection {
	primary := primarySurface(provider)
	view := &managementv1.ProviderView{
		ProviderId:           strings.TrimSpace(provider.GetProviderId()),
		DisplayName:          strings.TrimSpace(provider.GetDisplayName()),
		VendorId:             vendorIDFromSurface(primary),
		ProviderCredentialId: strings.TrimSpace(primary.GetProviderCredentialRef().GetProviderCredentialId()),
		ModelCatalog:         cloneProviderModelCatalog(primary.GetRuntime().GetCatalog()),
		Surfaces:             surfaceViewsFromProvider(provider),
	}
	return &ProviderProjection{value: view}
}

func surfaceViewsFromProvider(provider *providerv1.Provider) []*managementv1.ProviderSurfaceBindingView {
	surfaces := provider.GetSurfaces()
	items := make([]*managementv1.ProviderSurfaceBindingView, 0, len(surfaces))
	for _, surface := range surfaces {
		if surface == nil {
			continue
		}
		runtime := cloneProviderSurfaceRuntime(surface.GetRuntime())
		items = append(items, &managementv1.ProviderSurfaceBindingView{
			DisplayName:          surfaceDisplayName(surface),
			SurfaceId:            strings.TrimSpace(surface.GetSurfaceId()),
			ProviderCredentialId: strings.TrimSpace(surface.GetProviderCredentialRef().GetProviderCredentialId()),
			Runtime:              runtime,
			Status:               statusFromSurface(surface),
			VendorId:             vendorIDFromSurface(surface),
			ProviderId:           strings.TrimSpace(provider.GetProviderId()),
			ProviderDisplayName:  strings.TrimSpace(provider.GetDisplayName()),
		})
	}
	slices.SortFunc(items, compareSurfaces)
	return items
}

func statusFromSurface(surface *providerv1.ProviderSurfaceBinding) *managementv1.ProviderSurfaceBindingStatus {
	if surface == nil {
		return &managementv1.ProviderSurfaceBindingStatus{
			Phase:  managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG,
			Reason: "provider surface binding is nil",
		}
	}
	if err := providerv1.ValidateProviderSurfaceBinding(surface); err != nil {
		return &managementv1.ProviderSurfaceBindingStatus{
			Phase:  managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG,
			Reason: err.Error(),
		}
	}
	return &managementv1.ProviderSurfaceBindingStatus{
		Phase: managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY,
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

func (p *ProviderProjection) VendorID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetVendorId())
}

func (p *ProviderProjection) CredentialID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetProviderCredentialId())
}

func (p *ProviderProjection) SurfaceIDs() []string {
	if p == nil || p.value == nil {
		return nil
	}
	items := make([]string, 0, len(p.value.GetSurfaces()))
	for _, surface := range p.value.GetSurfaces() {
		surfaceID := strings.TrimSpace(surface.GetSurfaceId())
		if surfaceID != "" {
			items = append(items, surfaceID)
		}
	}
	return items
}

func (p *ProviderProjection) AuthKind() providerv1.ProviderSurfaceKind {
	surface := p.primarySurface()
	if surface == nil || surface.GetRuntime() == nil {
		return providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_UNSPECIFIED
	}
	return providerv1.RuntimeKind(surface.GetRuntime())
}

func (p *ProviderProjection) CLIID() string {
	surface := p.primarySurface()
	if surface == nil || surface.GetRuntime() == nil {
		return ""
	}
	return providerv1.RuntimeCLIID(surface.GetRuntime())
}

func (p *ProviderProjection) IconURL(vendorIcons, cliIcons map[string]string) string {
	switch p.AuthKind() {
	case providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI:
		return strings.TrimSpace(cliIcons[p.CLIID()])
	case providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API:
		return strings.TrimSpace(vendorIcons[p.VendorID()])
	default:
		return ""
	}
}

func (p *ProviderProjection) WithIconURL(iconURL string) *ProviderProjection {
	next := p.Proto()
	next.IconUrl = strings.TrimSpace(iconURL)
	return &ProviderProjection{value: next}
}

func (p *ProviderProjection) WithCredentialSubjectSummary(fields []*managementv1.CredentialSubjectSummaryFieldView) *ProviderProjection {
	next := p.Proto()
	if len(fields) == 0 {
		next.CredentialSubjectSummary = nil
		return &ProviderProjection{value: next}
	}
	next.CredentialSubjectSummary = make([]*managementv1.CredentialSubjectSummaryFieldView, 0, len(fields))
	for _, field := range fields {
		if field != nil {
			next.CredentialSubjectSummary = append(next.CredentialSubjectSummary, proto.Clone(field).(*managementv1.CredentialSubjectSummaryFieldView))
		}
	}
	return &ProviderProjection{value: next}
}

func (p *ProviderProjection) primarySurface() *managementv1.ProviderSurfaceBindingView {
	if p == nil || p.value == nil || len(p.value.GetSurfaces()) == 0 {
		return nil
	}
	return p.value.GetSurfaces()[0]
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

func surfaceDisplayName(surface *providerv1.ProviderSurfaceBinding) string {
	if surface == nil {
		return ""
	}
	if displayName := strings.TrimSpace(surface.GetRuntime().GetDisplayName()); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(surface.GetSurfaceId())
}

func primarySurface(provider *providerv1.Provider) *providerv1.ProviderSurfaceBinding {
	if provider == nil || len(provider.GetSurfaces()) == 0 {
		return &providerv1.ProviderSurfaceBinding{}
	}
	return provider.GetSurfaces()[0]
}

func vendorIDFromSurface(surface *providerv1.ProviderSurfaceBinding) string {
	if surface.GetSourceRef().GetKind() != providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR {
		return ""
	}
	return strings.TrimSpace(surface.GetSourceRef().GetId())
}
