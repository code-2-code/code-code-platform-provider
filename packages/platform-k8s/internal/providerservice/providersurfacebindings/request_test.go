package providersurfacebindings

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProtoToInstanceMapsProviderCredentialRef(t *testing.T) {
	instance, err := protoToProviderSurfaceBinding(&providerv1.ProviderSurfaceBinding{
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
			DisplayName: "Sample OpenAI Compatible",
			Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL,
			Access: &providerv1.ProviderSurfaceRuntime_Api{
				Api: &providerv1.ProviderAPISurfaceRuntime{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
					BaseUrl:  "https://api.example.com/v1",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("protoToProviderSurfaceBinding() error = %v", err)
	}
	if instance.GetSurfaceId() != "sample-openai-compatible" {
		t.Fatalf("SurfaceId = %q, want sample-openai-compatible", instance.GetSurfaceId())
	}
	if instance.GetProviderCredentialRef() == nil || instance.GetProviderCredentialRef().GetProviderCredentialId() != "sample-api-key" {
		t.Fatalf("ProviderCredentialRef = %#v, want provider credential id", instance.GetProviderCredentialRef())
	}
}
