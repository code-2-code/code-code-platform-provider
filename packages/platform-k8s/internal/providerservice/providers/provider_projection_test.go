package providers

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderProjectionUsesSurfaceEndpoints(t *testing.T) {
	view := providerProjectionFromProvider(repositoryTestProvider(), apiSurface("surface-a")).Proto()
	if got, want := len(view.GetEndpoints()), 1; got != want {
		t.Fatalf("endpoints len = %d, want %d", got, want)
	}
	if got, want := view.GetEndpoints()[0].GetApi().GetBaseUrl(), "https://api.example.com/v1"; got != want {
		t.Fatalf("endpoint base_url = %q, want %q", got, want)
	}
}

func TestProviderProjectionKeepsProviderModels(t *testing.T) {
	provider := repositoryTestProvider()
	provider.Models = []*providerv1.ProviderModel{{ProviderModelId: "model-a"}}

	view := providerProjectionFromProvider(provider, apiSurface("surface-a")).Proto()
	if got, want := view.GetModels()[0].GetProviderModelId(), "model-a"; got != want {
		t.Fatalf("provider model id = %q, want %q", got, want)
	}
}

func TestProviderProjectionDerivesCustomAPIKeyEndpointFromProvider(t *testing.T) {
	provider := repositoryTestProvider()
	provider.SurfaceId = "custom.api"
	provider.CustomApiKeySurface = &providerv1.CustomAPIKeySurface{
		BaseUrl:  "https://api.custom.example/v1",
		Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
	}

	view := providerProjectionFromProvider(provider, apiSurface("custom.api")).Proto()
	if got, want := len(view.GetEndpoints()), 1; got != want {
		t.Fatalf("endpoints len = %d, want %d", got, want)
	}
	if got, want := view.GetEndpoints()[0].GetApi().GetBaseUrl(), "https://api.custom.example/v1"; got != want {
		t.Fatalf("custom endpoint base_url = %q, want %q", got, want)
	}
}

func apiSurface(surfaceID string) *supportv1.Surface {
	return &supportv1.Surface{
		SurfaceId: surfaceID,
		Spec: &supportv1.Surface_Api{Api: &supportv1.ApiSurface{ApiEndpoints: []*supportv1.ApiEndpoint{{
			Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
			BaseUrl:  "https://api.example.com/v1",
		}}}},
	}
}
