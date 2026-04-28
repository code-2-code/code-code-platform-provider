package providerconnect

import (
	"encoding/json"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestSessionRecordJSONRoundTripPreservesTargetSnapshot(t *testing.T) {
	record, err := newSessionRecord(
		"session-1",
		newConnectTargetWithIDs(
			AddMethodCLIOAuth,
			"Codex",
			"openai",
			"codex",
			"codex",
			"credential-codex",
			"provider-codex",
			testProviderSurfaceBinding("codex", "codex"),
		),
		&credentialv1.OAuthAuthorizationSessionStatus{
			Phase:            credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_AWAITING_USER,
			AuthorizationUrl: "https://auth.example.com/device",
			UserCode:         "ABCD-EFGH",
			Message:          "Authorize this device",
		},
	)
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	decoded := &sessionRecord{}
	if err := json.Unmarshal(payload, decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	runtime, err := decoded.runtime()
	if err != nil {
		t.Fatalf("decoded.runtime() error = %v", err)
	}
	if got, want := decoded.TargetProviderID, "provider-codex"; got != want {
		t.Fatalf("target_provider_id = %q, want %q", got, want)
	}
	if got, want := decoded.ProviderSurfaceID, "codex"; got != want {
		t.Fatalf("provider_surface_id = %q, want %q", got, want)
	}
	if got, want := decoded.AuthorizationURL, "https://auth.example.com/device"; got != want {
		t.Fatalf("authorization_url = %q, want %q", got, want)
	}
	if got, want := providerv1.RuntimeKind(runtime), providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API; got != want {
		t.Fatalf("surface kind = %v, want %v", got, want)
	}
}
