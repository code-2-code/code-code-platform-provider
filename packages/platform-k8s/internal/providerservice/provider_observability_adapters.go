package providerservice

import (
	"context"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
)

type oauthObservabilityCapability struct {
	runner *providerobservability.OAuthObservabilityRunner
}

func (c oauthObservabilityCapability) OwnerKind() providerobservability.OwnerKind {
	return providerobservability.OwnerKindCLI
}

func (c oauthObservabilityCapability) Supports(surface *managementv1.ProviderSurfaceBindingView) (string, bool) {
	runtime := surfaceBindingRuntime(surface)
	if runtime == nil || runtime.GetCli() == nil {
		return "", false
	}
	cliID := strings.TrimSpace(runtime.GetCli().GetCliId())
	return cliID, cliID != ""
}

func (c oauthObservabilityCapability) ProbeProvider(ctx context.Context, target providerobservability.ProbeTarget, trigger providerobservability.Trigger) (*providerobservability.ProbeResult, error) {
	result, err := c.runner.ProbeProvider(ctx, target.ProviderID, oauthProbeTrigger(trigger))
	if err != nil || result == nil {
		return nil, err
	}
	ownerID := strings.TrimSpace(result.CLIID)
	if ownerID == "" {
		ownerID = target.OwnerID
	}
	return &providerobservability.ProbeResult{
		OwnerKind:                providerobservability.OwnerKindCLI,
		OwnerID:                  ownerID,
		ProviderID:               result.ProviderID,
		ProviderSurfaceBindingID: result.ProviderSurfaceBindingID,
		Outcome:                  providerobservability.ProbeOutcome(result.Outcome),
		Message:                  result.Message,
		Reason:                   result.Reason,
		LastAttemptAt:            result.LastAttemptAt,
		NextAllowedAt:            result.NextAllowedAt,
	}, nil
}

type vendorObservabilityCapability struct {
	runner *providerobservability.VendorObservabilityRunner
}

func (c vendorObservabilityCapability) OwnerKind() providerobservability.OwnerKind {
	return providerobservability.OwnerKindVendor
}

func (c vendorObservabilityCapability) Supports(surface *managementv1.ProviderSurfaceBindingView) (string, bool) {
	runtime := surfaceBindingRuntime(surface)
	if runtime == nil || runtime.GetApi() == nil {
		return "", false
	}
	vendorID := strings.TrimSpace(surface.GetVendorId())
	return vendorID, vendorID != ""
}

func (c vendorObservabilityCapability) ProbeProvider(ctx context.Context, target providerobservability.ProbeTarget, trigger providerobservability.Trigger) (*providerobservability.ProbeResult, error) {
	result, err := c.runner.ProbeProvider(ctx, target.ProviderID, vendorProbeTrigger(trigger))
	if err != nil || result == nil {
		return nil, err
	}
	ownerID := strings.TrimSpace(result.VendorID)
	if ownerID == "" {
		ownerID = target.OwnerID
	}
	return &providerobservability.ProbeResult{
		OwnerKind:                providerobservability.OwnerKindVendor,
		OwnerID:                  ownerID,
		ProviderID:               result.ProviderID,
		ProviderSurfaceBindingID: result.ProviderSurfaceBindingID,
		Outcome:                  providerobservability.ProbeOutcome(result.Outcome),
		Message:                  result.Message,
		Reason:                   result.Reason,
		LastAttemptAt:            result.LastAttemptAt,
		NextAllowedAt:            result.NextAllowedAt,
	}, nil
}

func surfaceBindingRuntime(surface *managementv1.ProviderSurfaceBindingView) *providerv1.ProviderSurfaceRuntime {
	if surface == nil || surface.GetRuntime() == nil {
		return nil
	}
	return surface.GetRuntime()
}

func oauthProbeTrigger(trigger providerobservability.Trigger) providerobservability.OAuthObservabilityProbeTrigger {
	switch trigger {
	case providerobservability.TriggerManual:
		return providerobservability.OAuthObservabilityProbeTriggerManual
	case providerobservability.TriggerConnect:
		return providerobservability.OAuthObservabilityProbeTriggerConnect
	default:
		return providerobservability.OAuthObservabilityProbeTriggerSchedule
	}
}

func vendorProbeTrigger(trigger providerobservability.Trigger) providerobservability.VendorObservabilityProbeTrigger {
	switch trigger {
	case providerobservability.TriggerManual:
		return providerobservability.VendorObservabilityProbeTriggerManual
	case providerobservability.TriggerConnect:
		return providerobservability.VendorObservabilityProbeTriggerConnect
	default:
		return providerobservability.VendorObservabilityProbeTriggerSchedule
	}
}
