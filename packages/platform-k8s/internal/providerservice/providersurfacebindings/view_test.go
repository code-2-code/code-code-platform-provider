package providersurfacebindings

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderSurfaceBindingToViewProjectsSurfaceBindingFields(t *testing.T) {
	surface := &providerv1.ProviderSurfaceBinding{
		SurfaceId: "sample-openai-compatible",
		SourceRef: &providerv1.ProviderSurfaceSourceRef{
			Kind:      providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR,
			Id:        "openai",
			SurfaceId: "sample-openai-compatible",
		},
		ProviderCredentialRef: &providerv1.ProviderCredentialRef{
			ProviderCredentialId: "sample-api-key",
		},
		Runtime: &providerv1.ProviderSurfaceRuntime{
			DisplayName: "Sample",
			Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL,
			Access: &providerv1.ProviderSurfaceRuntime_Api{
				Api: &providerv1.ProviderAPISurfaceRuntime{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
					BaseUrl:  "https://api.example.com/v1",
				},
			},
		},
	}

	view := providerSurfaceBindingToView(surface)
	if got, want := view.GetSurfaceId(), "sample-openai-compatible"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got, want := view.GetDisplayName(), "Sample"; got != want {
		t.Fatalf("display_name = %q, want %q", got, want)
	}
}
