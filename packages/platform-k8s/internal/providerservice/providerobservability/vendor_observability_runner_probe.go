package providerobservability

import (
	"context"
	"fmt"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/egressauth"
	"code-code.internal/platform-k8s/internal/platform/provideridentity"
)

func (r *VendorObservabilityRunner) probeProvider(
	ctx context.Context,
	providerID string,
	surface *providerv1.ProviderSurfaceBinding,
	trigger Trigger,
) (probeResult *ProbeResult, err error) {
	providerID = strings.TrimSpace(providerID)
	providerSurfaceBindingID := ""
	if surface != nil {
		providerSurfaceBindingID = strings.TrimSpace(surface.GetSurfaceId())
	}
	ctx, span := startVendorObservabilityProbeSpan(ctx, providerID, providerSurfaceBindingID, trigger)
	result := &ProbeResult{
		ProviderID:               providerID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		Outcome:                  ProbeOutcomeUnsupported,
	}
	defer func() {
		if probeResult != nil {
			result = probeResult
		}
		recordVendorObservabilitySpanError(span, err, "probe_failed")
		finishVendorObservabilityProbeSpan(span, result)
		span.End()
	}()
	if surface == nil || surface.GetRuntime() == nil {
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	runtime := surface.GetRuntime()
	if providerv1.RuntimeKind(runtime) != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	result.OwnerID = vendorOwnerID(surface)
	if result.OwnerID == "" || result.ProviderID == "" {
		result.Message = "vendor_id or provider_id is empty"
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	vendor, err := r.resolveVendor(ctx, result.OwnerID)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	activeQueryPolicy, supported, err := vendorActiveQueryPolicy(vendor, providerSurfaceBindingID)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	if !supported {
		result.Message = "vendor observability active_query is not configured"
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	collector, ok := r.collectors[activeQueryPolicy.CollectorID]
	if !ok {
		result.Message = fmt.Sprintf("collector %q is not registered", activeQueryPolicy.CollectorID)
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	now := r.now().UTC()
	if trigger != TriggerManual {
		if throttled, nextAllowedAt := r.throttled(result.ProviderID, providerSurfaceBindingID, now); throttled {
			result.Outcome = ProbeOutcomeThrottled
			result.Message = "operation is throttled by minimum interval"
			result.LastAttemptAt = timePointerCopy(&now)
			result.NextAllowedAt = timePointerCopy(&nextAllowedAt)
			r.metrics.record(result.OwnerID, result.ProviderID, trigger, result.Outcome, result.Reason, now, nextAllowedAt)
			return result, nil
		}
	}
	credentialID := vendorObservabilityCredentialID(result.ProviderID, surface, collector)
	if credentialID == "" {
		result.Message = "provider credential id is empty"
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	observabilityAuth, err := r.vendorObservabilityCredential(
		ctx,
		credentialID,
		&authv1.CredentialMaterialReadPolicyRef{
			Kind:        authv1.CredentialMaterialReadPolicyKind_CREDENTIAL_MATERIAL_READ_POLICY_KIND_VENDOR_ACTIVE_QUERY,
			OwnerId:     result.OwnerID,
			SurfaceId:   providerSurfaceBindingID,
			CollectorId: activeQueryPolicy.CollectorID,
		},
		activeQueryPolicy.MaterialReadFields,
	)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	httpClient, err := observabilityHTTPClient(ctx, observabilityEgressAuth{
		VendorID:                 result.OwnerID,
		ProviderID:               result.ProviderID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		RequestHeaderName:        "authorization",
		HeaderValuePrefix:        "Bearer",
		AuthAdapterID:            vendorObservabilityAuthAdapterID(collector),
	})
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	collectResult, collectErr := collector.Collect(ctx, ObservabilityCollectInput{
		OwnerID:                 result.OwnerID,
		ProviderID:               result.ProviderID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		CredentialID:             credentialID,
		SurfaceBaseURL:           strings.TrimSpace(providerv1.RuntimeBaseURL(runtime)),
		APIKey:                   observabilityCredentialToken(observabilityAuth),
		ObservabilityCredential:  observabilityAuth,
		CredentialBackfills:      activeQueryPolicy.CredentialBackfills,
		HTTPClient:               httpClient,
	})
	if collectErr != nil {
		if isObservabilityUnauthorizedError(collectErr) {
			result.Outcome = ProbeOutcomeAuthBlocked
			result.Reason = observabilityUnauthorizedReason(collectErr)
		} else {
			result.Outcome = ProbeOutcomeFailed
			result.Reason = observabilityFailureReasonFromError(collectErr)
		}
		result.Message = collectErr.Error()
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	if collectResult != nil {
		if err := mergeCredentialBackfills(ctx, r.credentialMerger, credentialID, activeQueryPolicy.CredentialBackfills, collectResult.CredentialBackfillValues); err != nil {
			result.Outcome = ProbeOutcomeFailed
			result.Reason = observabilityFailureReasonFromError(err)
			result.Message = err.Error()
			return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
		}
		r.metrics.recordCollectorValues(result.OwnerID, result.ProviderID, collectResult.GaugeRows)
	}
	result.Outcome = ProbeOutcomeExecuted
	result.Message = "operation completed"
	nextAllowedAfter := activeQueryPolicy.PollInterval
	if nextAllowedAfter < vendorObservabilityPendingBackoff {
		nextAllowedAfter = vendorObservabilityPendingBackoff
	}
	return r.recordProbeResult(result, trigger, now, nextAllowedAfter), nil
}

func (r *VendorObservabilityRunner) vendorObservabilityCredential(
	ctx context.Context,
	credentialID string,
	policyRef *authv1.CredentialMaterialReadPolicyRef,
	materialReadFields []string,
) (*credentialv1.ResolvedCredential, error) {
	credential := fakeSessionCredential(credentialID)
	if len(materialReadFields) == 0 {
		return credential, nil
	}
	values, err := readCredentialMaterialFields(ctx, r.credentialReader, credentialID, policyRef, materialReadFields)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return credential, nil
	}
	session := credential.GetSession()
	if session == nil {
		return credential, nil
	}
	if session.Values == nil {
		session.Values = map[string]string{}
	}
	for key, value := range values {
		session.Values[key] = value
	}
	return credential, nil
}

func vendorObservabilityCredentialID(providerID string, surface *providerv1.ProviderSurfaceBinding, collector ObservabilityCollector) string {
	if vendorObservabilityUsesObservabilityCredential(collector) {
		if credentialID := provideridentity.ObservabilityCredentialID(providerID); credentialID != "" {
			return credentialID
		}
	}
	if surface == nil {
		return ""
	}
	return strings.TrimSpace(surface.GetProviderCredentialRef().GetProviderCredentialId())
}

func vendorObservabilityUsesObservabilityCredential(collector ObservabilityCollector) bool {
	switch vendorObservabilityAuthAdapterID(collector) {
	case egressauth.AuthAdapterBearerSessionID, egressauth.AuthAdapterGoogleAIStudioSessionID, egressauth.AuthAdapterSessionCookieID:
		return true
	default:
		return false
	}
}

func vendorObservabilityAuthAdapterID(collector ObservabilityCollector) string {
	provider, ok := collector.(ObservabilityAuthAdapter)
	if !ok {
		return ""
	}
	return strings.TrimSpace(provider.AuthAdapterID())
}

func vendorOwnerID(surface *providerv1.ProviderSurfaceBinding) string {
	if surface == nil {
		return ""
	}
	source := surface.GetSourceRef()
	if source.GetKind() != providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR {
		return ""
	}
	return strings.TrimSpace(source.GetId())
}


