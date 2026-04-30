package providers

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderProjectionUsesRuntimeCLIIDAsProductInfoID(t *testing.T) {
	provider := repositoryTestProvider()
	provider.Runtime = &providerv1.ProviderSurfaceRuntime{
		DisplayName: "Codex",
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		Access: &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: "codex"},
		},
	}

	view := providerProjectionFromProvider(provider, providerSurfaceWithProduct("openai")).Proto()
	if got, want := view.GetProductInfoId(), "codex"; got != want {
		t.Fatalf("product_info_id = %q, want %q", got, want)
	}
}

func TestProviderProjectionUsesVendorEgressRulesetAsProductInfoID(t *testing.T) {
	provider := repositoryTestProvider()
	provider.Runtime.EgressRulesetId = "vendor.mistral"

	view := providerProjectionFromProvider(provider, providerSurfaceWithProduct("openai")).Proto()
	if got, want := view.GetProductInfoId(), "mistral"; got != want {
		t.Fatalf("product_info_id = %q, want %q", got, want)
	}
}

func TestProviderProjectionFallsBackToSurfaceProductInfoID(t *testing.T) {
	view := providerProjectionFromProvider(repositoryTestProvider(), providerSurfaceWithProduct("gemini")).Proto()
	if got, want := view.GetProductInfoId(), "gemini"; got != want {
		t.Fatalf("product_info_id = %q, want %q", got, want)
	}
}

func providerSurfaceWithProduct(productInfoID string) *providerv1.ProviderSurface {
	return &providerv1.ProviderSurface{
		SurfaceId: "surface-a",
		Kind:      providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
		SupportedCredentialKinds: []credentialv1.CredentialKind{
			credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY,
		},
		Spec: &providerv1.ProviderSurface_Api{
			Api: &providerv1.ProviderSurfaceAPISpec{
				ProductInfoId:      productInfoID,
				SupportedProtocols: []apiprotocolv1.Protocol{apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE},
			},
		},
	}
}
