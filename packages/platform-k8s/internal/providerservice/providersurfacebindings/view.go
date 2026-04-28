package providersurfacebindings

import (
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func providerSurfaceBindingToView(surface *providerv1.ProviderSurfaceBinding) *managementv1.ProviderSurfaceBindingView {
	return (&SurfaceBinding{value: surface}).View()
}

func (s *SurfaceBinding) View() *managementv1.ProviderSurfaceBindingView {
	if s == nil || s.value == nil {
		return &managementv1.ProviderSurfaceBindingView{}
	}
	runtime := proto.Clone(s.value.GetRuntime()).(*providerv1.ProviderSurfaceRuntime)
	return &managementv1.ProviderSurfaceBindingView{
		DisplayName:          s.DisplayName(),
		SurfaceId:            s.SurfaceID(),
		ProviderCredentialId: s.value.GetProviderCredentialRef().GetProviderCredentialId(),
		Runtime:              runtime,
		ProviderId:           s.ProviderID(),
	}
}

func surfaceBindingViewWithProvider(surface *SurfaceBinding, provider *providerv1.Provider) *managementv1.ProviderSurfaceBindingView {
	view := surface.View()
	if provider == nil {
		return view
	}
	view.ProviderId = provider.GetProviderId()
	view.ProviderDisplayName = provider.GetDisplayName()
	view.VendorId = vendorSourceID(surface.value)
	return view
}

func providerView(provider *providerv1.Provider) *managementv1.ProviderView {
	if provider == nil {
		return &managementv1.ProviderView{}
	}
	surfaces := make([]*managementv1.ProviderSurfaceBindingView, 0, len(provider.GetSurfaces()))
	for _, surface := range provider.GetSurfaces() {
		surfaces = append(surfaces, surfaceBindingViewWithProvider(&SurfaceBinding{value: surface, providerID: provider.GetProviderId()}, provider))
	}
	primary := firstSurface(provider)
	return &managementv1.ProviderView{
		ProviderId:           provider.GetProviderId(),
		DisplayName:          provider.GetDisplayName(),
		VendorId:             vendorSourceID(primary),
		ProviderCredentialId: primary.GetProviderCredentialRef().GetProviderCredentialId(),
		ModelCatalog:         primary.GetRuntime().GetCatalog(),
		Surfaces:             surfaces,
	}
}

func vendorSourceID(surface *providerv1.ProviderSurfaceBinding) string {
	if surface.GetSourceRef().GetKind() != providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR {
		return ""
	}
	return surface.GetSourceRef().GetId()
}

func firstSurface(provider *providerv1.Provider) *providerv1.ProviderSurfaceBinding {
	if provider == nil || len(provider.GetSurfaces()) == 0 {
		return &providerv1.ProviderSurfaceBinding{}
	}
	return provider.GetSurfaces()[0]
}
