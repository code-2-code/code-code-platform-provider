package providers

import (
	"strings"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/providerstate"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestUnmarshalProviderToleratesInvalidSurfaceChild(t *testing.T) {
	provider := repositoryTestProvider()
	provider.SurfaceId = ""
	payload, err := protojson.Marshal(provider)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	got, err := providerstate.NormalizeProviderForRead(provider.GetProviderId(), unmarshalProviderPayload(t, payload))
	if err != nil {
		t.Fatalf("NormalizeProviderForRead() error = %v", err)
	}
	status := providerProjectionFromProvider(got, nil).Proto().GetStatus()
	if got, want := status.GetPhase(), providerservicev1.ProviderPhase_PROVIDER_PHASE_INVALID_CONFIG; got != want {
		t.Fatalf("surface status phase = %v, want %v", got, want)
	}
	if !strings.Contains(status.GetReason(), "provider surface id is empty") {
		t.Fatalf("surface status reason = %q, want surface id validation error", status.GetReason())
	}
}

func TestMarshalProviderRejectsInvalidSurfaceChild(t *testing.T) {
	provider := repositoryTestProvider()
	provider.SurfaceId = ""

	if _, err := providerstate.NormalizeProviderForWrite("", provider); err == nil {
		t.Fatal("NormalizeProviderForWrite() error = nil, want validation error")
	}
}

func TestAccountFromProviderTreatsEmptyCatalogAsReady(t *testing.T) {
	view := providerProjectionFromProvider(repositoryTestProvider(), nil).Proto()
	status := view.GetStatus()
	if got, want := status.GetPhase(), providerservicev1.ProviderPhase_PROVIDER_PHASE_READY; got != want {
		t.Fatalf("surface status phase = %v, want %v", got, want)
	}
	if got := status.GetReason(); got != "" {
		t.Fatalf("surface status reason = %q, want empty", got)
	}
}

func unmarshalProviderPayload(t *testing.T, payload []byte) *providerv1.Provider {
	t.Helper()
	provider := &providerv1.Provider{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, provider); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return provider
}

func repositoryTestProvider() *providerv1.Provider {
	return &providerv1.Provider{
		ProviderId:  "provider-a",
		DisplayName: "Provider A",
		SurfaceId:   "definition-a",
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
	}
}
