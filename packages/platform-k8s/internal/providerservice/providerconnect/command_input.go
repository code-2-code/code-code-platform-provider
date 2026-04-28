package providerconnect

import (
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

// ConnectCommandInput carries one provider connect request into the owner.
type ConnectCommandInput struct {
	AddMethod   AddMethod
	DisplayName string
	VendorID    string
	CLIID       string
	APIKey      *APIKeyConnectInput
}

// APIKeyConnectInput carries API key onboarding material.
type APIKeyConnectInput struct {
	APIKey               string
	BaseURL              string
	Protocol             apiprotocolv1.Protocol
	SurfaceModelCatalogs []*ProviderSurfaceBindingModelCatalogInput
}

// ProviderSurfaceBindingModelCatalogInput carries one surface catalog override.
type ProviderSurfaceBindingModelCatalogInput struct {
	SurfaceID string
	Models    []*providerv1.ProviderModelCatalogEntry
}

func cloneAPIKeyConnectInput(input *APIKeyConnectInput) *APIKeyConnectInput {
	if input == nil {
		return nil
	}
	out := &APIKeyConnectInput{
		APIKey:   strings.TrimSpace(input.APIKey),
		BaseURL:  strings.TrimSpace(input.BaseURL),
		Protocol: input.Protocol,
	}
	if len(input.SurfaceModelCatalogs) > 0 {
		out.SurfaceModelCatalogs = make([]*ProviderSurfaceBindingModelCatalogInput, 0, len(input.SurfaceModelCatalogs))
		for _, item := range input.SurfaceModelCatalogs {
			if item == nil {
				continue
			}
			out.SurfaceModelCatalogs = append(out.SurfaceModelCatalogs, &ProviderSurfaceBindingModelCatalogInput{
				SurfaceID: strings.TrimSpace(item.SurfaceID),
				Models:    cloneProviderSurfaceBindingModels(item.Models),
			})
		}
	}
	return out
}

func cloneProviderSurfaceBindingModels(items []*providerv1.ProviderModelCatalogEntry) []*providerv1.ProviderModelCatalogEntry {
	if len(items) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderModelCatalogEntry, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, proto.Clone(item).(*providerv1.ProviderModelCatalogEntry))
	}
	return out
}
