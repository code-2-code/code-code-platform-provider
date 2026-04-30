package providerobservability

import (
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

func buildResponse(
	target ProbeTarget,
	result *ProbeResult,
) *managementv1.ProbeProviderObservabilityResponse {
	response := &managementv1.ProbeProviderObservabilityResponse{ProviderId: target.ProviderID}
	if result == nil {
		return response
	}
	if result.ProviderID != "" {
		response.ProviderId = result.ProviderID
	}

	response.Outcome = mapOutcome(result.Outcome)
	response.Message = result.Message
	response.NextAllowedAt = formatTime(result.NextAllowedAt)
	response.LastAttemptAt = formatTime(result.LastAttemptAt)
	return response
}

func mapOutcome(outcome ProbeOutcome) providerservicev1.ProviderOAuthObservabilityProbeOutcome {
	switch outcome {
	case ProbeOutcomeExecuted:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED
	case ProbeOutcomeThrottled:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_THROTTLED
	case ProbeOutcomeAuthBlocked:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_AUTH_BLOCKED
	case ProbeOutcomeUnsupported:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSUPPORTED
	case ProbeOutcomeFailed:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_FAILED
	default:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSPECIFIED
	}
}
