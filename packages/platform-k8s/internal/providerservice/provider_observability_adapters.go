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
	result, err := c.runner.ProbeProvider(ctx, target.ProviderID, trigger)
	if err != nil || result == nil {
		return nil, err
	}
	if result.OwnerKind == "" {
		result.OwnerKind = providerobservability.OwnerKindCLI
	}
	if strings.TrimSpace(result.OwnerID) == "" {
		result.OwnerID = target.OwnerID
	}
	return result, nil
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
	result, err := c.runner.ProbeProvider(ctx, target.ProviderID, trigger)
	if err != nil || result == nil {
		return nil, err
	}
	if result.OwnerKind == "" {
		result.OwnerKind = providerobservability.OwnerKindVendor
	}
	if strings.TrimSpace(result.OwnerID) == "" {
		result.OwnerID = target.OwnerID
	}
	return result, nil
}

func surfaceBindingRuntime(surface *managementv1.ProviderSurfaceBindingView) *providerv1.ProviderSurfaceRuntime {
	if surface == nil || surface.GetRuntime() == nil {
		return nil
	}
	return surface.GetRuntime()
}

