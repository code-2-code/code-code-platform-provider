package providerservice

import (
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func providerViewsToService(items []*managementv1.ProviderView) []*providerservicev1.ProviderView {
	out := make([]*providerservicev1.ProviderView, 0, len(items))
	for _, item := range items {
		if next := providerViewToService(item); next != nil {
			out = append(out, next)
		}
	}
	return out
}

func providerViewToService(view *managementv1.ProviderView) *providerservicev1.ProviderView {
	if view == nil {
		return nil
	}
	out := &providerservicev1.ProviderView{
		ProviderId:           view.GetProviderId(),
		DisplayName:          view.GetDisplayName(),
		SurfaceId:            view.GetSurfaceId(),
		ProviderCredentialId: view.GetProviderCredentialId(),
		ModelCatalog:         cloneProviderModelCatalog(view.GetModelCatalog()),
		ProductInfoId:        view.GetProductInfoId(),
		Status:               providerStatusToService(view.GetStatus()),
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
	}
	return out
}

func providerStatusToService(status *managementv1.ProviderStatus) *providerservicev1.ProviderStatus {
	if status == nil {
		return nil
	}
	return &providerservicev1.ProviderStatus{
		Phase:  status.GetPhase(),
		Reason: status.GetReason(),
	}
}

func cloneProviderModelCatalog(catalog *providerv1.ProviderModelCatalog) *providerv1.ProviderModelCatalog {
	if catalog == nil {
		return nil
	}
	return proto.Clone(catalog).(*providerv1.ProviderModelCatalog)
}

