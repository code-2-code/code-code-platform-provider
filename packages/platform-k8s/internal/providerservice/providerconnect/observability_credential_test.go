package providerconnect

import "testing"

func TestConnectTargetObservabilityCredentialOnlySupportsExplicitVendorSchema(t *testing.T) {
	t.Parallel()

	target := newConnectTargetWithIDs(
		AddMethodAPIKey,
		"MiniMax Provider",
		"minimax",
		"",
		"openai-compatible",
		"credential-minimax",
		"provider-minimax",
		testProviderSurfaceBinding("openai-compatible", "openai-compatible"),
	)

	if credential := target.ObservabilityCredential("session-minimax"); credential != nil {
		t.Fatalf("credential = %#v, want nil", credential)
	}
}
