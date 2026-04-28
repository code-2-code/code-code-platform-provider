package providerconnect

import (
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestNewOAuthFinalizePlanBuildsCreateProviderSurfaceBinding(t *testing.T) {
	record, err := newSessionRecord("session-1", newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		"Codex",
		"openai",
		"codex",
		"codex",
		"credential-codex",
		"provider-codex",
		&providerv1.ProviderSurfaceRuntime{
			DisplayName: "codex",
			Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
			Access: &providerv1.ProviderSurfaceRuntime_Cli{
				Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: "codex"},
			},
			Catalog: &providerv1.ProviderModelCatalog{
				Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
				Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
			},
		},
	), &credentialv1.OAuthAuthorizationSessionStatus{})
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}

	plan, err := newOAuthFinalizePlan(record, &credentialv1.OAuthAuthorizationSessionState{
		Spec: &credentialv1.OAuthAuthorizationSessionSpec{
			SessionId:          "session-1",
			TargetCredentialId: "credential-imported",
		},
	})
	if err != nil {
		t.Fatalf("newOAuthFinalizePlan() error = %v", err)
	}

	instance := plan.CreateProviderSurfaceBinding()
	if got, want := plan.CredentialID(), "credential-imported"; got != want {
		t.Fatalf("credential_id = %q, want %q", got, want)
	}
	if got, want := instance.GetProviderCredentialRef().GetProviderCredentialId(), "credential-imported"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
}

func TestOAuthFinalizePlanValidateExistingRejectsDifferentCredential(t *testing.T) {
	record, err := newSessionRecord("session-1", newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		"Codex",
		"openai",
		"codex",
		"codex",
		"credential-codex",
		"provider-codex",
		&providerv1.ProviderSurfaceRuntime{
			DisplayName: "codex",
			Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
			Access: &providerv1.ProviderSurfaceRuntime_Cli{
				Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: "codex"},
			},
			Catalog: &providerv1.ProviderModelCatalog{
				Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
				Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
			},
		},
	), &credentialv1.OAuthAuthorizationSessionStatus{})
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	plan, err := newOAuthFinalizePlan(record, &credentialv1.OAuthAuthorizationSessionState{
		Spec: &credentialv1.OAuthAuthorizationSessionSpec{
			SessionId:          "session-1",
			TargetCredentialId: "credential-imported",
		},
	})
	if err != nil {
		t.Fatalf("newOAuthFinalizePlan() error = %v", err)
	}

	err = plan.ValidateExisting(&ProviderSurfaceBindingView{
		SurfaceID:            "codex",
		ProviderCredentialID: "other-credential",
		ProviderID:           "provider-codex",
		ProviderDisplayName:  "Codex",
		VendorID:             "openai",
		Runtime:              plan.CreateProviderSurfaceBinding().GetRuntime(),
	})
	if err == nil {
		t.Fatal("ValidateExisting() error = nil, want conflict")
	}
}
