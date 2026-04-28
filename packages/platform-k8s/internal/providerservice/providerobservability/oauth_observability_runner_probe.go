package providerobservability

import (
	"context"
	"fmt"
	"strings"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/cliruntimeservice/cliversions"
	clioauth "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/oauth"
)

func (r *OAuthObservabilityRunner) probeProvider(
	ctx context.Context,
	providerID string,
	surface *providerv1.ProviderSurfaceBinding,
	trigger Trigger,
) (*ProbeResult, error) {
	providerID = strings.TrimSpace(providerID)
	providerSurfaceBindingID := ""
	if surface != nil {
		providerSurfaceBindingID = strings.TrimSpace(surface.GetSurfaceId())
	}
	result := &ProbeResult{
		ProviderID:               providerID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		Outcome:                  ProbeOutcomeUnsupported,
	}
	if surface == nil || surface.GetRuntime() == nil {
		return r.recordProbeResult(result, trigger, "", r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	runtime := surface.GetRuntime()
	if providerv1.RuntimeKind(runtime) != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI {
		return r.recordProbeResult(result, trigger, "", r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	cliID := strings.TrimSpace(providerv1.RuntimeCLIID(runtime))
	result.OwnerID = cliID
	if cliID == "" {
		return r.recordProbeResult(result, trigger, "", r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	activeQueryPolicy, supported, err := r.resolveActiveQueryPolicy(ctx, cliID)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, cliID, r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	if !supported {
		result.Message = "cli oauth observability active_query is not configured"
		return r.recordProbeResult(result, trigger, cliID, r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	collector, ok := r.collectors[activeQueryPolicy.CollectorID]
	if !ok {
		result.Message = fmt.Sprintf("collector %q is not registered", activeQueryPolicy.CollectorID)
		return r.recordProbeResult(result, trigger, cliID, r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}

	now := r.now().UTC()
	if trigger != TriggerManual {
		if throttled, nextAllowedAt := r.throttled(result.ProviderID, providerSurfaceBindingID, now); throttled {
			result.Outcome = ProbeOutcomeThrottled
			result.Message = "operation is throttled by minimum interval"
			result.LastAttemptAt = timePointerCopy(&now)
			result.NextAllowedAt = timePointerCopy(&nextAllowedAt)
			r.metrics.record(cliID, result.ProviderID, trigger, result.Outcome, result.Reason, now, nextAllowedAt)
			return result, nil
		}
	}

	credentialID := strings.TrimSpace(surface.GetProviderCredentialRef().GetProviderCredentialId())
	if credentialID == "" {
		result.Message = "provider credential id is empty"
		return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
	}
	if err := r.credentialFreshener.EnsureFresh(ctx, credentialID, oauthObservabilityEnsureFreshTTL); err != nil {
		result.Outcome = oauthObservabilityOutcomeForAuthError(err)
		result.Reason = oauthObservabilityReasonForOutcome(result.Outcome, err.Error())
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
	}

	var cli *supportv1.CLI
	if resolvedCLI, cliErr := r.cliSupport.Get(ctx, cliID); cliErr == nil {
		cli = resolvedCLI
	}
	var clientVersion string
	err = retryObservabilityTransientPlatform(ctx, func() error {
		var resolveErr error
		clientVersion, resolveErr = cliversions.Resolve(ctx, r.cliVersions, cliID)
		return resolveErr
	})
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
	}
	clientIdentity := clioauth.ResolveOAuthClientIdentity(cli, clientVersion)

	httpClient, err := observabilityHTTPClient(ctx, observabilityEgressAuth{
		CLIID:                    cliID,
		ProviderID:               result.ProviderID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		RequestHeaderName:        "authorization",
		HeaderValuePrefix:        "Bearer",
	})
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
	}
	materialValues, err := readCredentialMaterialFields(
		ctx,
		r.credentialReader,
		credentialID,
		&authv1.CredentialMaterialReadPolicyRef{
			Kind:        authv1.CredentialMaterialReadPolicyKind_CREDENTIAL_MATERIAL_READ_POLICY_KIND_CLI_OAUTH_ACTIVE_QUERY,
			OwnerId:     cliID,
			SurfaceId:   providerSurfaceBindingID,
			CollectorId: activeQueryPolicy.CollectorID,
		},
		activeQueryPolicy.MaterialReadFields,
	)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
	}
	collectResult, collectErr := collector.Collect(ctx, ObservabilityCollectInput{
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		CredentialID:             credentialID,
		AccessToken:              fakeOAuthCredential(credentialID).GetOauth().GetAccessToken(),
		HTTPClient:               httpClient,
		MaterialValues:           materialValues,
		CredentialBackfills:      activeQueryPolicy.CredentialBackfills,
		ClientVersion:            clientIdentity.ClientVersion,
		ModelCatalogUserAgent:    clientIdentity.ModelCatalogUserAgent,
		ObservabilityUserAgent:   clientIdentity.ObservabilityUserAgent,
	})
	if collectErr != nil {
		if isObservabilityUnauthorizedError(collectErr) {
			result.Outcome = ProbeOutcomeAuthBlocked
		} else {
			result.Outcome = ProbeOutcomeFailed
		}
		result.Reason = oauthObservabilityReasonForOutcome(result.Outcome, collectErr.Error())
		result.Message = collectErr.Error()
		return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
	}
	if collectResult != nil {
		if err := mergeCredentialBackfills(ctx, r.credentialMerger, credentialID, activeQueryPolicy.CredentialBackfills, collectResult.CredentialBackfillValues); err != nil {
			result.Outcome = ProbeOutcomeFailed
			result.Reason = observabilityFailureReasonFromError(err)
			result.Message = err.Error()
			return r.recordProbeResult(result, trigger, cliID, now, oauthObservabilityFailureBackoff), nil
		}
		r.metrics.recordCollectorValues(cliID, result.ProviderID, collectResult.GaugeRows)
	}

	result.Outcome = ProbeOutcomeExecuted
	result.Message = "operation completed"
	nextAllowedAfter := activeQueryPolicy.PollInterval
	if nextAllowedAfter < oauthObservabilityPendingBackoff {
		nextAllowedAfter = oauthObservabilityPendingBackoff
	}
	return r.recordProbeResult(result, trigger, cliID, now, nextAllowedAfter), nil
}



func oauthObservabilityOutcomeForAuthError(err error) ProbeOutcome {
	if err == nil {
		return ProbeOutcomeFailed
	}
	return oauthObservabilityOutcomeForAuthMessage(err.Error())
}

func oauthObservabilityOutcomeForAuthMessage(message string) ProbeOutcome {
	if oauthObservabilityIsAuthBlockedMessage(message) {
		return ProbeOutcomeAuthBlocked
	}
	return ProbeOutcomeFailed
}

func oauthObservabilityIsAuthBlockedMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "invalid_grant") ||
		strings.Contains(normalized, "invalid_token") ||
		strings.Contains(normalized, "unauthorized") ||
		strings.Contains(normalized, "access denied") ||
		strings.Contains(normalized, "expired") ||
		strings.Contains(normalized, "revoked") ||
		strings.Contains(normalized, "refresh_token is missing")
}

func oauthObservabilityReasonForOutcome(outcome ProbeOutcome, message string) string {
	if outcome == ProbeOutcomeAuthBlocked {
		return oauthObservabilityAuthBlockedReason(message)
	}
	return observabilityFailureReason(message)
}

func oauthObservabilityAuthBlockedReason(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(normalized, "invalid_grant"):
		return "INVALID_GRANT"
	case strings.Contains(normalized, "invalid_token"):
		return "INVALID_TOKEN"
	case strings.Contains(normalized, "refresh_token is missing"):
		return "REFRESH_TOKEN_MISSING"
	case strings.Contains(normalized, "revoked"):
		return "TOKEN_REVOKED"
	case strings.Contains(normalized, "expired"):
		return "TOKEN_EXPIRED"
	case strings.Contains(normalized, "access denied"):
		return "ACCESS_DENIED"
	case strings.Contains(normalized, "unauthorized"):
		return "UNAUTHORIZED"
	default:
		return "AUTH_BLOCKED"
	}
}
