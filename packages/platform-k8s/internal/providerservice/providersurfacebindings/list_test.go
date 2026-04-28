package providersurfacebindings

import (
	"context"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestListProviderSurfaceBindingsSkipsInvalidResource(t *testing.T) {
	t.Parallel()

	service, _ := newServiceForTest(
		testListProvider("provider-account-1", validProviderSurfaceBinding("provider-account-1", "instance-1")),
		testListProvider("provider-account-2", &providerv1.ProviderSurfaceBinding{SurfaceId: "broken-instance"}),
	)
	items, err := service.ListProviderSurfaceBindings(context.Background())
	if err != nil {
		t.Fatalf("ListProviderSurfaceBindings() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	if got, want := items[0].GetSurfaceId(), "instance-1"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
}

func testListProvider(providerID string, surface *providerv1.ProviderSurfaceBinding) *providerv1.Provider {
	return &providerv1.Provider{
		ProviderId:  providerID,
		DisplayName: "OpenAI Account",
		Surfaces:    []*providerv1.ProviderSurfaceBinding{surface},
	}
}

func validProviderSurfaceBinding(_ string, surfaceID string) *providerv1.ProviderSurfaceBinding {
	return &providerv1.ProviderSurfaceBinding{
		SurfaceId: surfaceID,
		SourceRef: &providerv1.ProviderSurfaceSourceRef{
			Kind:      providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR,
			Id:        "openai",
			SurfaceId: surfaceID,
		},
		ProviderCredentialRef: &providerv1.ProviderCredentialRef{
			ProviderCredentialId: "credential-openai",
		},
		Runtime: &providerv1.ProviderSurfaceRuntime{
			DisplayName: surfaceID,
			Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
			Access: &providerv1.ProviderSurfaceRuntime_Api{
				Api: &providerv1.ProviderAPISurfaceRuntime{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
					BaseUrl:  "https://example.com/v1",
				},
			},
		},
	}
}
