package providerobservability

import managementv1 "code-code.internal/go-contract/platform/management/v1"

func buildResponse(
	target ProbeTarget,
	result *ProbeResult,
) *managementv1.ProbeProviderObservabilityResponse {
	response := &managementv1.ProbeProviderObservabilityResponse{ProviderId: target.ProviderID}
	ownerKind := target.OwnerKind
	ownerID := target.OwnerID
	if result == nil {
		return response
	}
	if result.ProviderID != "" {
		response.ProviderId = result.ProviderID
	}
	if result.OwnerKind != "" {
		ownerKind = result.OwnerKind
	}
	if result.OwnerID != "" {
		ownerID = result.OwnerID
	}
	if ownerKind == OwnerKindCLI {
		response.CliId = ownerID
	}
	response.Outcome = mapOutcome(result.Outcome)
	response.Message = result.Message
	response.NextAllowedAt = formatTime(result.NextAllowedAt)
	response.LastAttemptAt = formatTime(result.LastAttemptAt)
	return response
}

func mapOutcome(outcome ProbeOutcome) managementv1.ProviderOAuthObservabilityProbeOutcome {
	switch outcome {
	case ProbeOutcomeExecuted:
		return managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED
	case ProbeOutcomeThrottled:
		return managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_THROTTLED
	case ProbeOutcomeAuthBlocked:
		return managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_AUTH_BLOCKED
	case ProbeOutcomeUnsupported:
		return managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSUPPORTED
	case ProbeOutcomeFailed:
		return managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_FAILED
	default:
		return managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSPECIFIED
	}
}
