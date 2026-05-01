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
	CLIID       string
	SurfaceID   string
	APIKey      *APIKeyConnectInput
}

// APIKeyConnectInput carries the pre-created credential reference for API key onboarding.
type APIKeyConnectInput struct {
	CredentialID  string
	BaseURL       string
	Protocol      apiprotocolv1.Protocol
	SurfaceModels []*SurfaceModelInput
}

// SurfaceModelInput carries provider models selected for one surface.
type SurfaceModelInput struct {
	SurfaceID string
	Models    []*providerv1.ProviderModel
}

func cloneAPIKeyConnectInput(input *APIKeyConnectInput) *APIKeyConnectInput {
	if input == nil {
		return nil
	}
	out := &APIKeyConnectInput{
		CredentialID: strings.TrimSpace(input.CredentialID),
		BaseURL:      strings.TrimSpace(input.BaseURL),
		Protocol:     input.Protocol,
	}
	if len(input.SurfaceModels) > 0 {
		out.SurfaceModels = make([]*SurfaceModelInput, 0, len(input.SurfaceModels))
		for _, item := range input.SurfaceModels {
			if item == nil {
				continue
			}
			out.SurfaceModels = append(out.SurfaceModels, &SurfaceModelInput{
				SurfaceID: strings.TrimSpace(item.SurfaceID),
				Models:    cloneProviderModels(item.Models),
			})
		}
	}
	return out
}

func cloneProviderModels(items []*providerv1.ProviderModel) []*providerv1.ProviderModel {
	if len(items) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderModel, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, proto.Clone(item).(*providerv1.ProviderModel))
	}
	return out
}
