package providerorchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/provideridentity"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultProviderProbeMinimumInterval = 5 * time.Minute

func (a *Activities) ProbePreflight(ctx context.Context, input ProviderProbePreflightInput) (*ProviderProbePreflightDecision, error) {
	if err := validateProbeKind(input.Kind); err != nil {
		return nil, err
	}
	provider, err := a.GetProvider(ctx, input.ProviderID)
	if err != nil {
		return nil, activityError(err)
	}
	surfaces, err := providersurfaces.NewService()
	if err != nil {
		return nil, activityError(err)
	}
	surface, err := surfaces.Get(ctx, provider.GetSurfaceId())
	if err != nil {
		return nil, activityError(err)
	}
	now := time.Now().UTC()
	manual := normalizeProbeTrigger(input.Trigger) == providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_MANUAL
	decision := &ProviderProbePreflightDecision{
		ProviderID: provider.GetProviderId(),
		Kind:       input.Kind,
		ProbeID:    providerProbeID(provider.GetProviderId(), input.Kind),
	}
	switch input.Kind {
	case providerProbeKindModelCatalog:
		decision.MinimumInterval = modelCatalogMinimumInterval(surface)
		if !modelCatalogProbeSupported(provider, surface) {
			decision.Outcome = "unsupported"
			decision.Message = "provider model catalog probe is unsupported"
			return decision, nil
		}
		if !manual && probeThrottled(provider.GetProbeStatus().GetModelCatalog(), now) {
			decision.Outcome = "throttled"
			decision.Message = "provider model catalog probe is waiting for its minimum interval"
			return decision, nil
		}
	case providerProbeKindQuota:
		profile := quotaProfile(surface)
		if profile == nil {
			decision.Outcome = "unsupported"
			decision.Message = "provider quota probe is unsupported"
			return decision, nil
		}
		query := profile.GetQuotaQuery()
		decision.MinimumInterval = quotaMinimumInterval(query)
		if !manual && probeThrottled(provider.GetProbeStatus().GetQuota(), now) {
			decision.Outcome = "throttled"
			decision.Message = "provider quota probe is waiting for its minimum interval"
			return decision, nil
		}
		if err := a.quotaCredentialPreflight(ctx, provider, query); err != nil {
			decision.Outcome = "auth_blocked"
			decision.Message = err.Error()
			return decision, nil
		}
	default:
		return nil, fmt.Errorf("platformk8s/providerorchestration: unsupported probe kind %q", input.Kind)
	}
	decision.ShouldRun = true
	decision.Outcome = "ready"
	decision.Message = "provider probe preflight passed"
	return decision, nil
}

func (a *Activities) RunQuotaProbeTask(ctx context.Context, input ProviderProbeTaskInput) (*managementv1.ProbeProviderObservabilityResponse, error) {
	input.Trigger = normalizeProbeTrigger(input.Trigger)
	decision, err := a.ProbePreflight(ctx, ProviderProbePreflightInput{ProviderID: input.ProviderID, Kind: providerProbeKindQuota, Trigger: input.Trigger})
	if err != nil {
		return nil, err
	}
	if !decision.ShouldRun {
		if decision.Outcome != "throttled" {
			state, err := a.RecordProbeStatus(ctx, probeStatusInput(decision, decision.Outcome, decision.Message, input.Trigger))
			if err != nil {
				return nil, err
			}
			response := quotaProbeResponseFromDecision(decision)
			applyQuotaProbeTiming(response, state)
			return response, nil
		}
		return quotaProbeResponseFromDecision(decision), nil
	}
	response, err := a.runQuotaProbe(ctx, input)
	if err != nil {
		_, _ = a.RecordProbeStatus(ctx, probeStatusInput(decision, "failed", "provider quota probe failed", input.Trigger))
		return nil, err
	}
	outcome := "executed"
	if response.GetOutcome() != providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSPECIFIED {
		outcome = response.GetOutcome().String()
	}
	message := firstNonEmptyString(response.GetMessage(), "provider quota probe completed")
	state, err := a.RecordProbeStatus(ctx, probeStatusInput(decision, outcome, message, input.Trigger))
	if err != nil {
		return nil, err
	}
	response.ProbeId = decision.ProbeID
	applyQuotaProbeTiming(response, state)
	return response, nil
}

func (a *Activities) runQuotaProbe(ctx context.Context, input ProviderProbeTaskInput) (*managementv1.ProbeProviderObservabilityResponse, error) {
	if a == nil || a.provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider client is nil")
	}
	response, err := a.provider.ProbeProviderObservability(ctx, &providerservicev1.ProbeProviderObservabilityRequest{
		ProviderId: strings.TrimSpace(input.ProviderID),
		Trigger:    input.Trigger,
	})
	if err != nil {
		return nil, activityError(err)
	}
	out := &managementv1.ProbeProviderObservabilityResponse{}
	if err := transcodeProto(response, out); err != nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: transcode quota probe response: %w", err)
	}
	return out, nil
}

func (a *Activities) RunModelCatalogProbeTask(ctx context.Context, input ProviderProbeTaskInput) (*managementv1.ProbeProviderModelCatalogResponse, error) {
	input.Trigger = normalizeProbeTrigger(input.Trigger)
	decision, err := a.ProbePreflight(ctx, ProviderProbePreflightInput{ProviderID: input.ProviderID, Kind: providerProbeKindModelCatalog, Trigger: input.Trigger})
	if err != nil {
		return nil, err
	}
	if !decision.ShouldRun {
		if decision.Outcome != "throttled" {
			if _, err := a.RecordProbeStatus(ctx, probeStatusInput(decision, decision.Outcome, decision.Message, input.Trigger)); err != nil {
				return nil, err
			}
		}
		return modelCatalogProbeResponseFromDecision(decision), nil
	}
	response, err := a.runModelCatalogProbe(ctx, input)
	if err != nil {
		_, _ = a.RecordProbeStatus(ctx, probeStatusInput(decision, "failed", "provider model catalog probe failed", input.Trigger))
		return nil, err
	}
	message := firstNonEmptyString(response.GetMessage(), "provider model catalog probe completed")
	if _, err := a.RecordProbeStatus(ctx, probeStatusInput(decision, "executed", message, input.Trigger)); err != nil {
		return nil, err
	}
	response.ProbeId = decision.ProbeID
	return response, nil
}

func (a *Activities) runModelCatalogProbe(ctx context.Context, input ProviderProbeTaskInput) (*managementv1.ProbeProviderModelCatalogResponse, error) {
	if a == nil || a.provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider client is nil")
	}
	response, err := a.provider.ProbeProviderModelCatalog(ctx, &providerservicev1.ProbeProviderModelCatalogRequest{
		ProviderId: strings.TrimSpace(input.ProviderID),
	})
	if err != nil {
		return nil, activityError(err)
	}
	out := &managementv1.ProbeProviderModelCatalogResponse{}
	if err := transcodeProto(response, out); err != nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: transcode model catalog probe response: %w", err)
	}
	return out, nil
}

func (a *Activities) RecordProbeStatus(ctx context.Context, input ProviderProbeStatusInput) (*providerv1.ProviderProbeRunState, error) {
	if a == nil || a.provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider client is nil")
	}
	providerID := strings.TrimSpace(input.ProviderID)
	if providerID == "" {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider id is empty")
	}
	now := time.Now().UTC()
	minimumInterval := input.MinimumInterval
	if minimumInterval <= 0 {
		minimumInterval = defaultProviderProbeMinimumInterval
	}
	state := &providerv1.ProviderProbeRunState{
		LastAttemptAt: timestamppb.New(now),
		NextAllowedAt: timestamppb.New(now.Add(minimumInterval)),
		ProbeId:       strings.TrimSpace(input.ProbeID),
		Outcome:       strings.TrimSpace(input.Outcome),
		Message:       strings.TrimSpace(input.Message),
	}
	_, err := a.provider.ApplyProviderProbeStatus(ctx, &providerservicev1.ApplyProviderProbeStatusRequest{
		ProviderId: providerID,
		ProbeKind:  providerProbeKindToProto(input.Kind),
		State:      state,
	})
	if err != nil {
		return nil, activityError(err)
	}
	recordProviderProbeMetric(input, now, now.Add(minimumInterval))
	return state, nil
}

func (a *Activities) ListProbeProviderIDs(ctx context.Context) ([]string, error) {
	if a == nil || a.provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider client is nil")
	}
	response, err := a.provider.ListProviders(ctx, &providerservicev1.ListProvidersRequest{})
	if err != nil {
		return nil, activityError(err)
	}
	out := make([]string, 0, len(response.GetItems()))
	for _, provider := range response.GetItems() {
		if providerID := strings.TrimSpace(provider.GetProviderId()); providerID != "" {
			out = append(out, providerID)
		}
	}
	return out, nil
}

func (a *Activities) quotaCredentialPreflight(ctx context.Context, provider *managementv1.ProviderView, query *observabilityv1.QuotaQueryCollection) error {
	if query == nil {
		return fmt.Errorf("provider quota probe is unsupported")
	}
	if !query.GetRequiresQuotaCredential() {
		if strings.TrimSpace(provider.GetProviderCredentialId()) == "" {
			return fmt.Errorf("provider data-plane credential is required for quota probe")
		}
		return nil
	}
	credentialID := provideridentity.ObservabilityCredentialID(provider.GetProviderId())
	if strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("provider quota credential is required")
	}
	if a == nil || a.auth == nil {
		return fmt.Errorf("provider quota credential status is unavailable")
	}
	response, err := a.auth.ListCredentials(ctx, &authv1.ListCredentialsRequest{})
	if err != nil {
		return activityError(err)
	}
	for _, credential := range response.GetItems() {
		if strings.TrimSpace(credential.GetCredentialId()) != credentialID {
			continue
		}
		if credential.GetStatus().GetMaterialReady() {
			return nil
		}
		return fmt.Errorf("provider quota credential is not ready")
	}
	return fmt.Errorf("provider quota credential is missing")
}

func modelCatalogProbeSupported(provider *managementv1.ProviderView, surface *supportv1.Surface) bool {
	if strings.TrimSpace(surface.GetModelCatalogProbeId()) != "" {
		return true
	}
	for _, endpoint := range provider.GetEndpoints() {
		if providerv1.EndpointBaseURL(endpoint) != "" && providerv1.EndpointProtocol(endpoint) != 0 {
			return true
		}
	}
	return false
}

func quotaProfile(surface *supportv1.Surface) *observabilityv1.ObservabilityProfile {
	if surface == nil {
		return nil
	}
	quotaProbeID := strings.TrimSpace(surface.GetQuotaProbeId())
	for _, profile := range surface.GetObservability().GetProfiles() {
		query := profile.GetQuotaQuery()
		if query == nil {
			continue
		}
		collectorID := strings.TrimSpace(query.GetCollectorId())
		if quotaProbeID == "" || collectorID == quotaProbeID {
			return profile
		}
	}
	return nil
}

func modelCatalogMinimumInterval(surface *supportv1.Surface) time.Duration {
	value := surface.GetModelCatalogProbeMinimumInterval().AsDuration()
	if value <= 0 {
		return defaultProviderProbeMinimumInterval
	}
	return value
}

func quotaMinimumInterval(query *observabilityv1.QuotaQueryCollection) time.Duration {
	value := query.GetMinimumPollInterval().AsDuration()
	if value <= 0 {
		return defaultProviderProbeMinimumInterval
	}
	return value
}

func probeThrottled(state *providerv1.ProviderProbeRunState, now time.Time) bool {
	if state == nil || state.GetNextAllowedAt() == nil {
		return false
	}
	nextAllowedAt := state.GetNextAllowedAt().AsTime()
	return !nextAllowedAt.IsZero() && now.Before(nextAllowedAt)
}

func providerProbeKindToProto(kind providerProbeKind) providerv1.ProviderProbeKind {
	switch kind {
	case providerProbeKindModelCatalog:
		return providerv1.ProviderProbeKind_PROVIDER_PROBE_KIND_MODEL_CATALOG
	case providerProbeKindQuota:
		return providerv1.ProviderProbeKind_PROVIDER_PROBE_KIND_QUOTA
	default:
		return providerv1.ProviderProbeKind_PROVIDER_PROBE_KIND_UNSPECIFIED
	}
}

func normalizeProbeTrigger(value providerservicev1.ProviderObservabilityProbeTrigger) providerservicev1.ProviderObservabilityProbeTrigger {
	if value == providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_UNSPECIFIED {
		return providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_MANUAL
	}
	return value
}

func probeStatusInput(
	decision *ProviderProbePreflightDecision,
	outcome string,
	message string,
	trigger providerservicev1.ProviderObservabilityProbeTrigger,
) ProviderProbeStatusInput {
	if decision == nil {
		return ProviderProbeStatusInput{Outcome: outcome, Message: message, Trigger: trigger}
	}
	return ProviderProbeStatusInput{
		ProviderID:      decision.ProviderID,
		Kind:            decision.Kind,
		ProbeID:         decision.ProbeID,
		Outcome:         outcome,
		Message:         message,
		MinimumInterval: decision.MinimumInterval,
		Trigger:         trigger,
	}
}

func applyQuotaProbeTiming(response *managementv1.ProbeProviderObservabilityResponse, state *providerv1.ProviderProbeRunState) {
	if response == nil || state == nil {
		return
	}
	response.LastAttemptAt = probeTimestampString(state.GetLastAttemptAt())
	response.NextAllowedAt = probeTimestampString(state.GetNextAllowedAt())
}

func probeTimestampString(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	timestamp := value.AsTime()
	if timestamp.IsZero() {
		return ""
	}
	return timestamp.UTC().Format(time.RFC3339)
}
