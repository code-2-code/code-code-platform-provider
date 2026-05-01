package providerconnect

import (
	"context"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
)

type apiKeyResolvedConnect struct {
	target *connectTarget
}

func newAPIKeyResolvedConnect(target *connectTarget) *apiKeyResolvedConnect {
	return &apiKeyResolvedConnect{target: target}
}

func (r *apiKeyResolvedConnect) Execute(
	ctx context.Context,
	apiKey string,
	runtime apiKeyConnectRuntime,
) (*apiKeyConnectResult, error) {
	switch {
	case r == nil:
		return nil, domainerror.NewValidation("platformk8s/providerconnect: api key connect target is required")
	case r.target != nil:
		return newCustomAPIKeyConnectExecution(r.target, apiKey).Execute(ctx, runtime)
	default:
		return nil, domainerror.NewValidation("platformk8s/providerconnect: api key connect target is required")
	}
}

type providerConnectAPIKeyResolutionRuntime struct {
	support providerConnectSupport
	queries *providerConnectQueries
}

func newProviderConnectAPIKeyResolutionRuntime(
	support providerConnectSupport,
	queries *providerConnectQueries,
) providerConnectAPIKeyResolutionRuntime {
	return providerConnectAPIKeyResolutionRuntime{
		support: support,
		queries: queries,
	}
}

func (r providerConnectAPIKeyResolutionRuntime) Resolve(
	ctx context.Context,
	command *ConnectCommand,
) (*apiKeyResolvedConnect, error) {
	if command.IsCustomAPIKey() {
		return r.ResolveCustom(ctx, command)
	}
	if command.IsSurfaceAPIKey() {
		return r.ResolveSurface(ctx, command)
	}
	return nil, domainerror.NewValidation("platformk8s/providerconnect: surface_id is required for API key connect")
}

func (r providerConnectAPIKeyResolutionRuntime) ResolveCustom(
	ctx context.Context,
	command *ConnectCommand,
) (*apiKeyResolvedConnect, error) {
	displayName := command.DisplayNameOr("Custom API Key")
	surfaceModels, err := newSurfaceModelSet(command.SurfaceModels())
	if err != nil {
		return nil, err
	}
	candidate, err := newCustomAPIKeyCandidate(displayName, command.APIKeyInput(), surfaceModels)
	if err != nil {
		return nil, err
	}
	if err := surfaceModels.ValidateAllMatched(); err != nil {
		return nil, err
	}
	target, err := candidate.APIKeyTarget(displayName)
	if err != nil {
		return nil, err
	}
	return newAPIKeyResolvedConnect(target), nil
}

func (r providerConnectAPIKeyResolutionRuntime) ResolveSurface(
	ctx context.Context,
	command *ConnectCommand,
) (*apiKeyResolvedConnect, error) {
	displayName := command.DisplayNameOr(command.SurfaceID())
	surfaceModels, err := newSurfaceModelSet(command.SurfaceModels())
	if err != nil {
		return nil, err
	}
	surface, err := r.queries.LoadSurfaceMetadata(ctx, command.SurfaceID())
	if err != nil {
		return nil, err
	}
	endpoints := providersurfaces.Endpoints(surface.value)
	if len(endpoints) == 0 {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface %q does not expose an API endpoint", command.SurfaceID())
	}
	baseURL, usesCustomBaseURL, err := resolveSurfaceAPIKeyBaseURL(endpoints[0].GetApi().GetBaseUrl(), command.APIKeyInput().BaseURL)
	if err != nil {
		return nil, err
	}
	endpoint := endpoints[0]
	if usesCustomBaseURL {
		endpoint = &providerv1.ProviderEndpoint{
			Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_API,
			Shape: &providerv1.ProviderEndpoint_Api{Api: &providerv1.ProviderApiEndpoint{
				Protocol: endpoints[0].GetApi().GetProtocol(),
				BaseUrl:  baseURL,
			}},
		}
	}
	candidate, err := newConnectProviderCandidate(
		surface.SurfaceID(),
		endpoint,
		surfaceModels.Models(surface.SurfaceID(), nil),
		"platformk8s/providerconnect: invalid provider surface",
	)
	if err != nil {
		return nil, err
	}
	if usesCustomBaseURL {
		candidate.customAPIKeySurface = &providerv1.CustomAPIKeySurface{
			BaseUrl:  baseURL,
			Protocol: candidate.Endpoint().GetApi().GetProtocol(),
		}
	}
	if !usesCustomBaseURL {
		if err := surface.ValidateCandidate(candidate, credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY); err != nil {
			return nil, err
		}
	}
	if err := surfaceModels.ValidateAllMatched(); err != nil {
		return nil, err
	}
	target, err := candidate.APIKeyTarget(displayName)
	if err != nil {
		return nil, err
	}
	return newAPIKeyResolvedConnect(target), nil
}
