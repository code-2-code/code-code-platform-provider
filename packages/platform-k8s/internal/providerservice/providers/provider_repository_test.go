package providers

import (
	"strings"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestUnmarshalProviderToleratesInvalidSurfaceChild(t *testing.T) {
	provider := repositoryTestProvider()
	provider.Surfaces[0].SourceRef.SurfaceId = ""
	payload, err := protojson.Marshal(provider)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	got, err := unmarshalProvider(provider.GetProviderId(), payload)
	if err != nil {
		t.Fatalf("unmarshalProvider() error = %v", err)
	}
	status := providerProjectionFromProvider(got).Proto().GetSurfaces()[0].GetStatus()
	if got, want := status.GetPhase(), managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG; got != want {
		t.Fatalf("surface status phase = %v, want %v", got, want)
	}
	if !strings.Contains(status.GetReason(), "provider surface source surface id is empty") {
		t.Fatalf("surface status reason = %q, want source surface id validation error", status.GetReason())
	}
}

func TestMarshalProviderRejectsInvalidSurfaceChild(t *testing.T) {
	provider := repositoryTestProvider()
	provider.Surfaces[0].SourceRef.SurfaceId = ""

	if _, _, _, err := marshalProvider(provider); err == nil {
		t.Fatal("marshalProvider() error = nil, want validation error")
	}
}

func TestAccountFromProviderTreatsEmptyCatalogAsReady(t *testing.T) {
	view := providerProjectionFromProvider(repositoryTestProvider()).Proto()
	status := view.GetSurfaces()[0].GetStatus()
	if got, want := status.GetPhase(), managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY; got != want {
		t.Fatalf("surface status phase = %v, want %v", got, want)
	}
	if got := status.GetReason(); got != "" {
		t.Fatalf("surface status reason = %q, want empty", got)
	}
}

func repositoryTestProvider() *providerv1.Provider {
	return &providerv1.Provider{
		ProviderId:  "provider-a",
		DisplayName: "Provider A",
		Surfaces: []*providerv1.ProviderSurfaceBinding{{
			SurfaceId: "definition-a",
			Runtime: &providerv1.ProviderSurfaceRuntime{
				DisplayName: "Surface A",
				Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
				Access: &providerv1.ProviderSurfaceRuntime_Api{
					Api: &providerv1.ProviderAPISurfaceRuntime{
						Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
						BaseUrl:  "https://api.example.com/v1",
					},
				},
			},
			SourceRef: &providerv1.ProviderSurfaceSourceRef{
				Kind:      providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR,
				Id:        "vendor-a",
				SurfaceId: "definition-a",
			},
		}},
	}
}
