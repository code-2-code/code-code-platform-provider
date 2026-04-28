package providersurfacebindings

import (
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func protoToProviderSurfaceBinding(input *providerv1.ProviderSurfaceBinding) (*providerv1.ProviderSurfaceBinding, error) {
	surface, err := newProviderSurfaceBinding(input)
	if err != nil {
		return nil, err
	}
	return surface.Proto(), nil
}

func normalizeProviderSurfaceBinding(surface *providerv1.ProviderSurfaceBinding) {
	if surface == nil {
		return
	}
	if surface.Runtime == nil {
		surface.Runtime = &providerv1.ProviderSurfaceRuntime{}
	}
	if surface.Runtime.DisplayName == "" {
		surface.Runtime.DisplayName = surface.GetSurfaceId()
	}
}

func cloneProviderSurfaceBindingProto(input *providerv1.ProviderSurfaceBinding) *providerv1.ProviderSurfaceBinding {
	if input == nil {
		return &providerv1.ProviderSurfaceBinding{}
	}
	return proto.Clone(input).(*providerv1.ProviderSurfaceBinding)
}

func providerSurfaceBindingDisplayName(input *providerv1.ProviderSurfaceBinding) string {
	if input == nil {
		return ""
	}
	next := cloneProviderSurfaceBindingProto(input)
	normalizeProviderSurfaceBinding(next)
	return next.GetRuntime().GetDisplayName()
}
