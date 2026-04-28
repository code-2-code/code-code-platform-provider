package providerservice

import (
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func providerSurfaceBindingFromUpsertRequest(request *providerservicev1.UpsertProviderSurfaceBindingRequest) *providerv1.ProviderSurfaceBinding {
	if request == nil {
		return nil
	}
	surface := &providerv1.ProviderSurfaceBinding{
		SurfaceId: request.GetSurfaceId(),
		Runtime:   cloneProviderSurfaceBinding(request.GetRuntime()),
	}
	if request.GetProviderCredentialId() != "" {
		surface.ProviderCredentialRef = &providerv1.ProviderCredentialRef{ProviderCredentialId: request.GetProviderCredentialId()}
	}
	if surface.GetRuntime() != nil && surface.GetRuntime().GetDisplayName() == "" {
		surface.Runtime.DisplayName = request.GetDisplayName()
	}
	return surface
}

func cloneProviderSurfaceBinding(runtime *providerv1.ProviderSurfaceRuntime) *providerv1.ProviderSurfaceRuntime {
	if runtime == nil {
		return nil
	}
	return proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
}
