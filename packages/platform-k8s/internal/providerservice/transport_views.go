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
		Models:               cloneProviderModels(view.GetModels()),
		Endpoints:            cloneProviderEndpoints(view.GetEndpoints()),
		Observability:        providerObservabilityToService(view.GetObservability()),
		ProbeStatus:          cloneProviderProbeStatus(view.GetProbeStatus()),
		ProductInfoId:        view.GetProductInfoId(),
		Status:               providerStatusToService(view.GetStatus()),
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
	}
	return out
}

func providerObservabilityToService(view *managementv1.ProviderObservabilityView) *providerservicev1.ProviderObservabilityView {
	if view == nil {
		return nil
	}
	return &providerservicev1.ProviderObservabilityView{
		ObservabilityPolicyId: view.GetObservabilityPolicyId(),
		Status: &providerservicev1.ProviderObservabilityStatus{
			SupportsQuota:          view.GetStatus().GetSupportsQuota(),
			SupportsModelUsage:     view.GetStatus().GetSupportsModelUsage(),
			SupportsAccountSummary: view.GetStatus().GetSupportsAccountSummary(),
		},
	}
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

func cloneProviderModels(models []*providerv1.ProviderModel) []*providerv1.ProviderModel {
	if len(models) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderModel, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		out = append(out, proto.Clone(model).(*providerv1.ProviderModel))
	}
	return out
}

func cloneProviderEndpoints(endpoints []*providerv1.ProviderEndpoint) []*providerv1.ProviderEndpoint {
	if len(endpoints) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint == nil {
			continue
		}
		out = append(out, proto.Clone(endpoint).(*providerv1.ProviderEndpoint))
	}
	return out
}

func cloneProviderProbeStatus(status *providerv1.ProviderProbeStatus) *providerv1.ProviderProbeStatus {
	if status == nil {
		return nil
	}
	return proto.Clone(status).(*providerv1.ProviderProbeStatus)
}
