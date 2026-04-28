package providerobservability

import (
	"testing"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
)

func TestCredentialBackfillUpdatesKeepsOnlyDeclaredValues(t *testing.T) {
	t.Parallel()

	updates, err := credentialBackfillUpdates([]CredentialBackfillRule{
		{
			RuleID:            "project-id",
			Source:            observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_COLLECTOR_OUTPUT,
			SourceName:        "project_id",
			TargetMaterialKey: "project_id",
			Required:          true,
		},
		{
			RuleID:            "tier-name",
			Source:            observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_COLLECTOR_OUTPUT,
			SourceName:        "tier_name",
			TargetMaterialKey: "tier_name",
		},
	}, map[string]string{
		"project_id": " workspacecli-489315 ",
		"tier_name":  "Google AI Pro",
		"set-cookie": "must-not-persist",
	})
	if err != nil {
		t.Fatalf("credentialBackfillUpdates() error = %v", err)
	}
	if got, want := updates["project_id"], "workspacecli-489315"; got != want {
		t.Fatalf("project_id = %q, want %q", got, want)
	}
	if got, want := updates["tier_name"], "Google AI Pro"; got != want {
		t.Fatalf("tier_name = %q, want %q", got, want)
	}
	if _, exists := updates["set-cookie"]; exists {
		t.Fatal("undeclared set-cookie value was included")
	}
}

func TestCredentialBackfillUpdatesRequiresDeclaredValues(t *testing.T) {
	t.Parallel()

	_, err := credentialBackfillUpdates([]CredentialBackfillRule{{
		RuleID:            "project-id",
		Source:            observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_COLLECTOR_OUTPUT,
		SourceName:        "project_id",
		TargetMaterialKey: "project_id",
		Required:          true,
	}}, map[string]string{})
	if err == nil {
		t.Fatal("credentialBackfillUpdates() error = nil, want required backfill error")
	}
}
