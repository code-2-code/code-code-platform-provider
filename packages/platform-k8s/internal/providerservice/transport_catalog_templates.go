package providerservice

import (
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)


func templateViewsToService(items []*managementv1.TemplateView) []*providerservicev1.TemplateView {
	out := make([]*providerservicev1.TemplateView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &providerservicev1.TemplateView{
			TemplateId:         item.GetTemplateId(),
			DisplayName:        item.GetDisplayName(),
			Vendor:             item.GetVendor(),
			Protocol:           item.GetProtocol(),
			DefaultBaseUrl:     item.GetDefaultBaseUrl(),
			DefaultModels:      append([]string(nil), item.GetDefaultModels()...),
			RequiresCredential: item.GetRequiresCredential(),
		})
	}
	return out
}

func applyTemplateRequestToManagement(request *providerservicev1.ApplyTemplateRequest) *managementv1.ApplyTemplateRequest {
	if request == nil {
		return nil
	}
	return &managementv1.ApplyTemplateRequest{
		TemplateId:           request.GetTemplateId(),
		Namespace:            request.GetNamespace(),
		DisplayName:          request.GetDisplayName(),
		ProviderId:           request.GetProviderId(),
		AllowedModelIds:      append([]string(nil), request.GetAllowedModelIds()...),
		ProviderCredentialId: request.GetProviderCredentialId(),
	}
}

func applyTemplateResultToService(result *managementv1.ApplyTemplateResult) *providerservicev1.ApplyTemplateResult {
	if result == nil {
		return nil
	}
	return &providerservicev1.ApplyTemplateResult{
		TemplateId:   result.GetTemplateId(),
		Namespace:    result.GetNamespace(),
		DisplayName:  result.GetDisplayName(),
		ProviderId:   result.GetProviderId(),
		AppliedKinds: append([]string(nil), result.GetAppliedKinds()...),
	}
}
