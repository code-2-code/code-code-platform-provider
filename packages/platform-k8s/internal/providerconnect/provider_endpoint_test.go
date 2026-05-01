package providerconnect

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestConnectTargetProviderPersistsSurfaceAndModelsOnly(t *testing.T) {
	target := newConnectTargetWithIDs(
		AddMethodAPIKey,
		"OpenAI Compatible",
		"",
		"openai-compatible",
		"credential-1",
		"provider-1",
		[]*providerv1.ProviderModel{{ProviderModelId: "gpt-4.1-mini"}},
	)

	provider := target.Provider("credential-1")
	if got, want := provider.GetSurfaceId(), "openai-compatible"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got := len(provider.GetModels()); got != 1 {
		t.Fatalf("models len = %d, want 1", got)
	}
}

func TestNewCustomAPIKeyCandidateUsesDerivedEndpoint(t *testing.T) {
	candidate, err := newCustomAPIKeyCandidate("custom", &APIKeyConnectInput{
		BaseURL:  "https://api.example.test/v1",
		Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
	}, &surfaceModelSet{})
	if err != nil {
		t.Fatalf("newCustomAPIKeyCandidate() error = %v", err)
	}
	if got, want := candidate.SurfaceID(), "custom.api"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	endpoint := candidate.Endpoint()
	if endpoint.GetType() != providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_API {
		t.Fatalf("endpoint type = %v, want API", endpoint.GetType())
	}
	if got, want := endpoint.GetApi().GetBaseUrl(), "https://api.example.test/v1"; got != want {
		t.Fatalf("base_url = %q, want %q", got, want)
	}
	target, err := candidate.APIKeyTarget("custom")
	if err != nil {
		t.Fatalf("APIKeyTarget() error = %v", err)
	}
	provider := target.Provider("credential-custom")
	if got, want := provider.GetCustomApiKeySurface().GetBaseUrl(), "https://api.example.test/v1"; got != want {
		t.Fatalf("custom_api_key_surface.base_url = %q, want %q", got, want)
	}
}

func testCLIOAuthSessionTarget(_ string) *connectTarget {
	return newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		"Codex",
		"codex",
		"codex",
		"credential-codex",
		"provider-codex",
		nil,
	)
}
