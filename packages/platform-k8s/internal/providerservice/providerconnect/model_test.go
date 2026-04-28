package providerconnect

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestNewConnectPlanAssignsSharedIdentityToTargets(t *testing.T) {
	targetA := newConnectTargetWithIDs(
		AddMethodAPIKey,
		"OpenAI Provider",
		"openai",
		"",
		"openai-compatible",
		"",
		"",
		testProviderSurfaceBinding("openai-compatible", "OpenAI Compatible"),
	)
	targetB := newConnectTargetWithIDs(
		AddMethodAPIKey,
		"OpenAI Provider",
		"openai",
		"",
		"anthropic",
		"",
		"",
		testProviderSurfaceBinding("anthropic", "Anthropic"),
	)

	plan, err := newConnectPlan("OpenAI Provider", "openai", []*connectTarget{targetA, targetB})
	if err != nil {
		t.Fatalf("newConnectPlan() error = %v", err)
	}
	if got, want := len(plan.Targets), 2; got != want {
		t.Fatalf("len(targets) = %d, want %d", got, want)
	}
	if plan.TargetCredentialID == "" || plan.TargetProviderID == "" {
		t.Fatalf("shared ids = (%q, %q), want non-empty", plan.TargetCredentialID, plan.TargetProviderID)
	}
	if got, want := plan.Targets[0].TargetCredentialID, plan.TargetCredentialID; got != want {
		t.Fatalf("target credential id = %q, want %q", got, want)
	}
	if got, want := plan.Targets[1].TargetProviderID, plan.TargetProviderID; got != want {
		t.Fatalf("target provider id = %q, want %q", got, want)
	}
	if got, want := plan.Targets[1].SurfaceID, "anthropic"; got != want {
		t.Fatalf("target surface id = %q, want %q", got, want)
	}
}

func TestConnectTargetProviderUsesSurfaceID(t *testing.T) {
	target := newConnectTargetWithIDs(
		AddMethodAPIKey,
		"OpenAI Provider",
		"openai",
		"",
		"openai-compatible",
		"credential-openai",
		"provider-openai",
		testProviderSurfaceBinding("openai-compatible", "openai-compatible"),
	)

	provider := target.Provider("credential-openai")
	surface := provider.GetSurfaces()[0]
	if got, want := surface.GetSurfaceId(), "openai-compatible"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got, want := surface.GetSourceRef().GetSurfaceId(), "openai-compatible"; got != want {
		t.Fatalf("source surface_id = %q, want %q", got, want)
	}
	if got, want := surface.GetRuntime().GetApi().GetBaseUrl(), "https://api.example.com/v1"; got != want {
		t.Fatalf("base_url = %q, want %q", got, want)
	}
	if err := providerv1.ValidateProvider(provider); err != nil {
		t.Fatalf("ValidateProvider() error = %v", err)
	}
}

func TestNewSessionRecordProjectsOAuthStatusAndTargetRoundTrip(t *testing.T) {
	runtime := testCLISurfaceRuntime("codex", "codex")
	runtime.Catalog = &providerv1.ProviderModelCatalog{
		Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
		Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
	}
	target := newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		"Codex",
		"openai",
		"codex",
		"codex",
		"credential-codex",
		"provider-codex",
		runtime,
	)

	record, err := newSessionRecord("session-1", target, &credentialv1.OAuthAuthorizationSessionStatus{
		Phase:            credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_AWAITING_USER,
		AuthorizationUrl: "https://auth.example.com/device",
		UserCode:         "ABCD-EFGH",
		Message:          "Authorize this device",
	})
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	if got, want := record.Phase, SessionPhaseAwaitingUser; got != want {
		t.Fatalf("phase = %v, want %v", got, want)
	}
	if got, want := record.AuthorizationURL, "https://auth.example.com/device"; got != want {
		t.Fatalf("authorization_url = %q, want %q", got, want)
	}

	decoded, err := decodeProviderSurfaceRuntime(record.Runtime)
	if err != nil {
		t.Fatalf("decodeProviderSurfaceBinding() error = %v", err)
	}
	instance := record.target(decoded).ProviderSurfaceBinding("credential-imported")
	if got, want := instance.GetProviderCredentialRef().GetProviderCredentialId(), "credential-imported"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
}

func TestNewCLIReauthorizationTargetBuildsOAuthSessionTarget(t *testing.T) {
	target, err := newCLIReauthorizationTarget(&ProviderView{
		ProviderID:           "provider-codex",
		DisplayName:          "Codex Provider",
		ProviderCredentialID: "credential-codex",
		VendorID:             "openai",
		Surfaces: []*ProviderSurfaceBindingView{{
			SurfaceID: "codex",
			Runtime:   testCLISurfaceRuntime("codex", "codex"),
		}},
	})
	if err != nil {
		t.Fatalf("newCLIReauthorizationTarget() error = %v", err)
	}
	if got, want := target.CLIID, "codex"; got != want {
		t.Fatalf("cli_id = %q, want %q", got, want)
	}
	if got, want := target.TargetCredentialID, "credential-codex"; got != want {
		t.Fatalf("target_credential_id = %q, want %q", got, want)
	}
	if got, want := target.TargetProviderID, "provider-codex"; got != want {
		t.Fatalf("target_provider_id = %q, want %q", got, want)
	}
	if got, want := target.SurfaceID, "codex"; got != want {
		t.Fatalf("target surface_id = %q, want %q", got, want)
	}
}

func testProviderSurfaceBinding(_ string, displayName string) *providerv1.ProviderSurfaceRuntime {
	runtime := testAPISurfaceRuntime(displayName, apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, "https://api.example.com/v1")
	runtime.Origin = providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL
	return runtime
}
