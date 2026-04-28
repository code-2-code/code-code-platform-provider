package providerconnect

import (
	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func testAPISurfaceRuntime(displayName string, protocol apiprotocolv1.Protocol, baseURL string) *providerv1.ProviderSurfaceRuntime {
	return &providerv1.ProviderSurfaceRuntime{
		DisplayName: displayName,
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		Access: &providerv1.ProviderSurfaceRuntime_Api{
			Api: &providerv1.ProviderAPISurfaceRuntime{
				Protocol: protocol,
				BaseUrl:  baseURL,
			},
		},
	}
}

func testCLISurfaceRuntime(displayName, cliID string) *providerv1.ProviderSurfaceRuntime {
	return &providerv1.ProviderSurfaceRuntime{
		DisplayName: displayName,
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		Access: &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: cliID},
		},
	}
}

func testProviderSurface(
	surfaceID string,
	kind providerv1.ProviderSurfaceKind,
	credentialKinds []credentialv1.CredentialKind,
	protocols ...apiprotocolv1.Protocol,
) *providerv1.ProviderSurface {
	surface := &providerv1.ProviderSurface{
		SurfaceId:                surfaceID,
		DisplayName:              surfaceID,
		Kind:                     kind,
		SupportedCredentialKinds: credentialKinds,
	}
	if kind == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
		surface.Api = &providerv1.ProviderSurfaceAPISpec{SupportedProtocols: protocols}
	}
	return surface
}
