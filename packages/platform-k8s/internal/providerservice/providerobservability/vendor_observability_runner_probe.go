package providerobservability

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/egressauth"
	"code-code.internal/platform-k8s/internal/platform/outboundhttp"
	"code-code.internal/platform-k8s/internal/platform/provideridentity"
)

func (r *VendorObservabilityRunner) probeProvider(
	ctx context.Context,
	providerID string,
	surface *providerv1.ProviderSurfaceBinding,
	trigger VendorObservabilityProbeTrigger,
) (probeResult *VendorObservabilityProbeResult, err error) {
	providerID = strings.TrimSpace(providerID)
	providerSurfaceBindingID := ""
	if surface != nil {
		providerSurfaceBindingID = strings.TrimSpace(surface.GetSurfaceId())
	}
	ctx, span := startVendorObservabilityProbeSpan(ctx, providerID, providerSurfaceBindingID, trigger)
	result := &VendorObservabilityProbeResult{
		ProviderID:               providerID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		Outcome:                  VendorObservabilityProbeOutcomeUnsupported,
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
	result.VendorID = vendorOwnerID(surface)
	if result.VendorID == "" || result.ProviderID == "" {
		result.Message = "vendor_id or provider_id is empty"
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	vendor, err := r.resolveVendor(ctx, result.VendorID)
	if err != nil {
		result.Outcome = VendorObservabilityProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	activeQueryPolicy, supported, err := vendorActiveQueryPolicy(vendor, providerSurfaceBindingID)
	if err != nil {
		result.Outcome = VendorObservabilityProbeOutcomeFailed
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
	if trigger != VendorObservabilityProbeTriggerManual {
		if throttled, nextAllowedAt := r.throttled(result.ProviderID, providerSurfaceBindingID, now); throttled {
			result.Outcome = VendorObservabilityProbeOutcomeThrottled
			result.Message = "operation is throttled by minimum interval"
			result.LastAttemptAt = timePointerCopy(&now)
			result.NextAllowedAt = timePointerCopy(&nextAllowedAt)
			r.metrics.record(result.VendorID, result.ProviderID, trigger, result.Outcome, result.Reason, now, nextAllowedAt)
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
			OwnerId:     result.VendorID,
			SurfaceId:   providerSurfaceBindingID,
			CollectorId: activeQueryPolicy.CollectorID,
		},
		activeQueryPolicy.MaterialReadFields,
	)
	if err != nil {
		result.Outcome = VendorObservabilityProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	httpClient, err := r.httpClient(ctx, observabilityEgressAuth{
		VendorID:                 result.VendorID,
		ProviderID:               result.ProviderID,
		ProviderSurfaceBindingID: providerSurfaceBindingID,
		RequestHeaderName:        "authorization",
		HeaderValuePrefix:        "Bearer",
		AuthAdapterID:            vendorObservabilityAuthAdapterID(collector),
	})
	if err != nil {
		result.Outcome = VendorObservabilityProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	collectResult, collectErr := collector.Collect(ctx, VendorObservabilityCollectInput{
		VendorID:                 result.VendorID,
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
		if isVendorObservabilityUnauthorizedError(collectErr) {
			result.Outcome = VendorObservabilityProbeOutcomeAuthBlocked
			result.Reason = vendorObservabilityUnauthorizedReason(collectErr)
		} else {
			result.Outcome = VendorObservabilityProbeOutcomeFailed
			result.Reason = observabilityFailureReasonFromError(collectErr)
		}
		result.Message = collectErr.Error()
		return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
	}
	if collectResult != nil {
		if err := mergeCredentialBackfills(ctx, r.credentialMerger, credentialID, activeQueryPolicy.CredentialBackfills, collectResult.CredentialBackfillValues); err != nil {
			result.Outcome = VendorObservabilityProbeOutcomeFailed
			result.Reason = observabilityFailureReasonFromError(err)
			result.Message = err.Error()
			return r.recordProbeResult(result, trigger, now, vendorObservabilityFailureBackoff), nil
		}
		r.metrics.recordCollectorValues(result.VendorID, result.ProviderID, collectResult.GaugeRows)
	}
	result.Outcome = VendorObservabilityProbeOutcomeExecuted
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
	values, err := r.readCredentialMaterialFields(ctx, credentialID, policyRef, materialReadFields)
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

func vendorObservabilityCredentialID(providerID string, surface *providerv1.ProviderSurfaceBinding, collector VendorObservabilityCollector) string {
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

func vendorObservabilityUsesObservabilityCredential(collector VendorObservabilityCollector) bool {
	switch vendorObservabilityAuthAdapterID(collector) {
	case egressauth.AuthAdapterBearerSessionID, egressauth.AuthAdapterGoogleAIStudioSessionID, egressauth.AuthAdapterSessionCookieID:
		return true
	default:
		return false
	}
}

func vendorObservabilityAuthAdapterID(collector VendorObservabilityCollector) string {
	provider, ok := collector.(VendorObservabilityAuthAdapter)
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

func (r *VendorObservabilityRunner) httpClient(ctx context.Context, auth observabilityEgressAuth) (*http.Client, error) {
	client, err := outboundhttp.NewClientFactory().NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return withObservabilityEgressAuth(client, auth), nil
}
