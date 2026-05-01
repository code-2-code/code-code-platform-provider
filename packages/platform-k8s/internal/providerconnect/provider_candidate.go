package providerconnect

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces/registry"
	"google.golang.org/protobuf/proto"
)

type connectProviderCandidate struct {
	surfaceID           string
	endpoint            *providerv1.ProviderEndpoint
	models              []*providerv1.ProviderModel
	customAPIKeySurface *providerv1.CustomAPIKeySurface
}

func newCustomAPIKeyCandidate(
	displayName string,
	material *APIKeyConnectInput,
	surfaceModels *surfaceModelSet,
) (*connectProviderCandidate, error) {
	_ = displayName
	if material == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: api key material is required")
	}
	surfaceID := registry.SurfaceIDCustomAPIKey
	endpoint := &providerv1.ProviderEndpoint{
		Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_API,
		Shape: &providerv1.ProviderEndpoint_Api{Api: &providerv1.ProviderApiEndpoint{
			Protocol: material.Protocol,
			BaseUrl:  strings.TrimSpace(material.BaseURL),
		}},
	}
	candidate, err := newConnectProviderCandidate(surfaceID, endpoint, surfaceModels.Models(surfaceID, nil), "platformk8s/providerconnect: invalid custom provider")
	if err != nil {
		return nil, err
	}
	candidate.customAPIKeySurface = &providerv1.CustomAPIKeySurface{
		BaseUrl:  strings.TrimSpace(material.BaseURL),
		Protocol: material.Protocol,
	}
	return candidate, nil
}

func newCLIOAuthCandidate(displayName, cliID, surfaceID string) (*connectProviderCandidate, error) {
	_ = displayName
	endpoint := &providerv1.ProviderEndpoint{
		Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_CLI,
		Shape: &providerv1.ProviderEndpoint_Cli{Cli: &providerv1.ProviderCliEndpoint{
			CliId: strings.TrimSpace(cliID),
		}},
	}
	return newConnectProviderCandidate(surfaceID, endpoint, nil, "platformk8s/providerconnect: invalid cli provider")
}

func newConnectProviderCandidate(surfaceID string, endpoint *providerv1.ProviderEndpoint, models []*providerv1.ProviderModel, message string) (*connectProviderCandidate, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return nil, domainerror.NewValidation("%s: surface_id is required", message)
	}
	if err := providerv1.ValidateProviderEndpoint(endpoint); err != nil {
		return nil, domainerror.NewValidation("%s: %v", message, err)
	}
	return &connectProviderCandidate{
		surfaceID: surfaceID,
		endpoint:  proto.Clone(endpoint).(*providerv1.ProviderEndpoint),
		models:    cloneProviderModels(models),
	}, nil
}

func (c *connectProviderCandidate) SurfaceID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.surfaceID)
}

func (c *connectProviderCandidate) Endpoint() *providerv1.ProviderEndpoint {
	if c == nil || c.endpoint == nil {
		return nil
	}
	return proto.Clone(c.endpoint).(*providerv1.ProviderEndpoint)
}

func (c *connectProviderCandidate) Models() []*providerv1.ProviderModel {
	if c == nil {
		return nil
	}
	return cloneProviderModels(c.models)
}

func (c *connectProviderCandidate) CustomAPIKeySurface() *providerv1.CustomAPIKeySurface {
	if c == nil || c.customAPIKeySurface == nil {
		return nil
	}
	return proto.Clone(c.customAPIKeySurface).(*providerv1.CustomAPIKeySurface)
}
