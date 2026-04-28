package providerservice

import (
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

func vendorViewsToService(items []*managementv1.VendorView) []*providerservicev1.VendorView {
	out := make([]*providerservicev1.VendorView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &providerservicev1.VendorView{
			VendorId:    item.GetVendorId(),
			DisplayName: item.GetDisplayName(),
			IconUrl:     item.GetIconUrl(),
			Aliases:     append([]string(nil), item.GetAliases()...),
		})
	}
	return out
}

func cliDefinitionViewsToService(items []*managementv1.CLIDefinitionView) []*providerservicev1.CLIDefinitionView {
	out := make([]*providerservicev1.CLIDefinitionView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &providerservicev1.CLIDefinitionView{
			CliId:           item.GetCliId(),
			DisplayName:     item.GetDisplayName(),
			IconUrl:         item.GetIconUrl(),
			WebsiteUrl:      item.GetWebsiteUrl(),
			ContainerImages: cliContainerImagesToService(item.GetContainerImages()),
			Description:     item.GetDescription(),
			Capabilities:    cliDefinitionCapabilitiesToService(item.GetCapabilities()),
		})
	}
	return out
}

func cliContainerImagesToService(items []*managementv1.CLIContainerImageView) []*providerservicev1.CLIContainerImageView {
	out := make([]*providerservicev1.CLIContainerImageView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &providerservicev1.CLIContainerImageView{
			ExecutionClass: item.GetExecutionClass(),
			Image:          item.GetImage(),
			CpuRequest:     item.GetCpuRequest(),
			MemoryRequest:  item.GetMemoryRequest(),
		})
	}
	return out
}

func cliDefinitionCapabilitiesToService(view *managementv1.CLIDefinitionCapabilityView) *providerservicev1.CLIDefinitionCapabilityView {
	if view == nil {
		return nil
	}
	return &providerservicev1.CLIDefinitionCapabilityView{
		SupportsStreaming:    view.GetSupportsStreaming(),
		SupportsApprovalMode: view.GetSupportsApprovalMode(),
		SupportedProtocols:   append([]string(nil), view.GetSupportedProtocols()...),
	}
}

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
