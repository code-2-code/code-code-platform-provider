package providerobservability

import (
	"context"
	"fmt"
	"strings"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
)

func (r *OAuthObservabilityRunner) mergeCredentialBackfills(
	ctx context.Context,
	credentialID string,
	rules []CredentialBackfillRule,
	values map[string]string,
) error {
	if r == nil {
		return nil
	}
	return mergeCredentialBackfills(ctx, r.credentialMerger, credentialID, rules, values)
}

func mergeCredentialBackfills(
	ctx context.Context,
	merger CredentialMaterialValueMerger,
	credentialID string,
	rules []CredentialBackfillRule,
	values map[string]string,
) error {
	if merger == nil || len(rules) == 0 || len(values) == 0 {
		return nil
	}
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return fmt.Errorf("providerobservability: credential id is empty")
	}
	updates, err := credentialBackfillUpdates(rules, values)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	return merger.MergeCredentialMaterialValues(ctx, credentialID, updates)
}

func credentialBackfillUpdates(rules []CredentialBackfillRule, values map[string]string) (map[string]string, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	updates := map[string]string{}
	for _, rule := range rules {
		if rule.Source == observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_UNSPECIFIED {
			continue
		}
		sourceName := strings.TrimSpace(rule.SourceName)
		targetKey := strings.TrimSpace(rule.TargetMaterialKey)
		if sourceName == "" || targetKey == "" {
			continue
		}
		value := strings.TrimSpace(values[sourceName])
		if value == "" {
			if rule.Required {
				return nil, fmt.Errorf("providerobservability: required credential backfill %q is empty", rule.RuleID)
			}
			continue
		}
		updates[targetKey] = value
	}
	if len(updates) == 0 {
		return nil, nil
	}
	return updates, nil
}
