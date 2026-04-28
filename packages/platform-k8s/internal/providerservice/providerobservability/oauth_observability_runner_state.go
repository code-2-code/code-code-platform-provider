package providerobservability

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

type activeQueryPolicySpec struct {
	PollInterval        time.Duration
	CollectorID         string
	CredentialBackfills []CredentialBackfillRule
	MaterialReadFields  []string
}

type CredentialBackfillRule struct {
	RuleID            string
	Source            observabilityv1.CredentialBackfillSource
	SourceName        string
	TargetMaterialKey string
	Required          bool
	Readable          bool
}

func (r *OAuthObservabilityRunner) resolveActiveQueryPolicy(ctx context.Context, cliID string) (activeQueryPolicySpec, bool, error) {
	var cli *supportv1.CLI
	err := retryObservabilityTransientPlatform(ctx, func() error {
		var getErr error
		cli, getErr = r.cliSupport.Get(ctx, strings.TrimSpace(cliID))
		return getErr
	})
	if err != nil {
		if observabilityTransientPlatformError(err) {
			return activeQueryPolicySpec{}, false, err
		}
		return activeQueryPolicySpec{}, false, nil
	}
	return activeQueryPolicy(cli)
}

func activeQueryPolicy(cli *supportv1.CLI) (activeQueryPolicySpec, bool, error) {
	if cli == nil || cli.GetOauth() == nil || cli.GetOauth().GetObservability() == nil {
		return activeQueryPolicySpec{}, false, nil
	}
	var (
		pollInterval time.Duration
		collectorID  string
		backfills    []CredentialBackfillRule
		readFields   []string
	)
	for _, profile := range cli.GetOauth().GetObservability().GetProfiles() {
		if profile == nil || profile.GetActiveQuery() == nil {
			continue
		}
		activeQuery := profile.GetActiveQuery()
		currentInterval := activeQuery.GetMinimumPollInterval().AsDuration()
		if currentInterval <= 0 {
			continue
		}
		if pollInterval == 0 || currentInterval < pollInterval {
			pollInterval = currentInterval
		}
		currentCollectorID := strings.TrimSpace(activeQuery.GetCollectorId())
		if currentCollectorID == "" {
			currentCollectorID = strings.TrimSpace(cli.GetCliId())
		}
		if collectorID != "" && currentCollectorID != "" && collectorID != currentCollectorID {
			return activeQueryPolicySpec{}, false, fmt.Errorf("providerobservability: cli oauth active query collector conflict: %q vs %q", collectorID, currentCollectorID)
		}
		if currentCollectorID != "" {
			collectorID = currentCollectorID
		}
		backfills = append(backfills, activeQueryCredentialBackfills(activeQuery)...)
		readFields = append(readFields, activeQueryMaterialReadFields(activeQuery)...)
	}
	if pollInterval <= 0 {
		return activeQueryPolicySpec{}, false, nil
	}
	if collectorID == "" {
		collectorID = strings.TrimSpace(cli.GetCliId())
	}
	return activeQueryPolicySpec{
		PollInterval:        pollInterval,
		CollectorID:         collectorID,
		CredentialBackfills: backfills,
		MaterialReadFields:  sortedUniqueStrings(readFields),
	}, true, nil
}

func activeQueryCredentialBackfills(activeQuery *observabilityv1.ActiveQueryCollection) []CredentialBackfillRule {
	if activeQuery == nil || len(activeQuery.GetCredentialBackfills()) == 0 {
		return nil
	}
	items := make([]CredentialBackfillRule, 0, len(activeQuery.GetCredentialBackfills()))
	for _, rule := range activeQuery.GetCredentialBackfills() {
		if rule == nil {
			continue
		}
		items = append(items, CredentialBackfillRule{
			RuleID:            strings.TrimSpace(rule.GetRuleId()),
			Source:            rule.GetSource(),
			SourceName:        strings.TrimSpace(rule.GetSourceName()),
			TargetMaterialKey: strings.TrimSpace(rule.GetTargetMaterialKey()),
			Required:          rule.GetRequired(),
			Readable:          rule.GetReadable(),
		})
	}
	return items
}

func activeQueryMaterialReadFields(activeQuery *observabilityv1.ActiveQueryCollection) []string {
	if activeQuery == nil {
		return nil
	}
	fields := append([]string(nil), activeQuery.GetMaterialReadFields()...)
	for _, rule := range activeQuery.GetCredentialBackfills() {
		if rule == nil || !rule.GetReadable() {
			continue
		}
		fields = append(fields, rule.GetTargetMaterialKey())
	}
	return sortedUniqueStrings(fields)
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func (r *OAuthObservabilityRunner) throttled(providerID string, providerSurfaceBindingID string, now time.Time) (bool, time.Time) {
	stateKey := oauthObservabilityStateKey(providerID, providerSurfaceBindingID)
	if stateKey == "" {
		return false, time.Time{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.states[stateKey]
	if !ok || state.nextAllowedAt.IsZero() {
		return false, time.Time{}
	}
	if now.Before(state.nextAllowedAt) {
		return true, state.nextAllowedAt
	}
	return false, state.nextAllowedAt
}

func (r *OAuthObservabilityRunner) nextAllowed(providerID string, providerSurfaceBindingID string) time.Time {
	stateKey := oauthObservabilityStateKey(providerID, providerSurfaceBindingID)
	if stateKey == "" {
		return time.Time{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.states[stateKey]
	if !ok {
		return time.Time{}
	}
	return state.nextAllowedAt
}

func (r *OAuthObservabilityRunner) recordProbeResult(
	result *OAuthObservabilityProbeResult,
	trigger OAuthObservabilityProbeTrigger,
	cliID string,
	now time.Time,
	backoff time.Duration,
) *OAuthObservabilityProbeResult {
	if result == nil {
		return nil
	}
	if strings.TrimSpace(result.CLIID) == "" {
		result.CLIID = strings.TrimSpace(cliID)
	}
	if backoff <= 0 {
		backoff = oauthObservabilityFailureBackoff
	}
	nextAllowedAt := now.Add(backoff)
	stateKey := oauthObservabilityStateKey(result.ProviderID, result.ProviderSurfaceBindingID)
	if stateKey == "" {
		stateKey = oauthObservabilityStateKey("", result.ProviderSurfaceBindingID)
	}
	r.mu.Lock()
	r.states[stateKey] = oauthObservabilityState{
		lastAttemptAt: now,
		nextAllowedAt: nextAllowedAt,
	}
	r.mu.Unlock()
	result.LastAttemptAt = timePointerCopy(&now)
	result.NextAllowedAt = timePointerCopy(&nextAllowedAt)
	r.metrics.record(result.CLIID, result.ProviderID, trigger, result.Outcome, result.Reason, now, nextAllowedAt)
	return result
}

func oauthObservabilityStateKey(providerID string, providerSurfaceBindingID string) string {
	if providerID := strings.TrimSpace(providerID); providerID != "" {
		return "provider:" + providerID
	}
	if surfaceID := strings.TrimSpace(providerSurfaceBindingID); surfaceID != "" {
		return "surface:" + surfaceID
	}
	return ""
}
