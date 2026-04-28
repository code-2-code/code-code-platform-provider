package providerobservability

import (
	"context"
	"fmt"
	"strings"
	"time"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

func (r *VendorObservabilityRunner) resolveVendor(ctx context.Context, vendorID string) (*supportv1.Vendor, error) {
	var vendor *supportv1.Vendor
	err := retryObservabilityTransientPlatform(ctx, func() error {
		var getErr error
		vendor, getErr = r.vendorSupport.Get(ctx, strings.TrimSpace(vendorID))
		return getErr
	})
	if err != nil {
		if observabilityTransientPlatformError(err) {
			return nil, err
		}
		return nil, nil
	}
	return vendor, nil
}

func vendorActiveQueryPolicy(vendor *supportv1.Vendor, surfaceID string) (activeQueryPolicySpec, bool, error) {
	if vendor == nil {
		return activeQueryPolicySpec{}, false, nil
	}
	var (
		pollInterval time.Duration
		collectorID  string
		backfills    []CredentialBackfillRule
		readFields   []string
	)
	for _, capability := range vendorObservabilityCapabilities(vendor, surfaceID) {
		for _, profile := range capability.GetProfiles() {
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
				continue
			}
			if collectorID != "" && collectorID != currentCollectorID {
				return activeQueryPolicySpec{}, false, fmt.Errorf("providerobservability: vendor active query collector conflict: %q vs %q", collectorID, currentCollectorID)
			}
			collectorID = currentCollectorID
			backfills = append(backfills, activeQueryCredentialBackfills(activeQuery)...)
			readFields = append(readFields, activeQueryMaterialReadFields(activeQuery)...)
		}
	}
	if pollInterval <= 0 {
		return activeQueryPolicySpec{}, false, nil
	}
	if collectorID == "" {
		collectorID = strings.TrimSpace(vendor.GetVendor().GetVendorId())
	}
	return activeQueryPolicySpec{
		PollInterval:        pollInterval,
		CollectorID:         collectorID,
		CredentialBackfills: backfills,
		MaterialReadFields:  sortedUniqueStrings(readFields),
	}, true, nil
}

func vendorObservabilityCapabilities(vendor *supportv1.Vendor, surfaceID string) []*observabilityv1.ObservabilityCapability {
	surfaceID = strings.TrimSpace(surfaceID)
	out := []*observabilityv1.ObservabilityCapability{}
	for _, binding := range vendor.GetProviderBindings() {
		if binding == nil || binding.GetObservability() == nil {
			continue
		}
		if surfaceID != "" && !vendorBindingHasSurface(binding, surfaceID) {
			continue
		}
		out = append(out, binding.GetObservability())
	}
	return out
}

func vendorBindingHasSurface(binding *supportv1.VendorProviderBinding, surfaceID string) bool {
	for _, template := range binding.GetSurfaceTemplates() {
		if strings.TrimSpace(template.GetSurfaceId()) == surfaceID {
			return true
		}
	}
	return false
}

func (r *VendorObservabilityRunner) recordProbeResult(
	result *ProbeResult,
	trigger Trigger,
	now time.Time,
	backoff time.Duration,
) *ProbeResult {
	if result == nil {
		return nil
	}
	r.logVendorObservabilityProbeFailure(result)
	return r.recordState(result, trigger, now, backoff)
}

