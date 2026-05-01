package providerorchestration

import (
	"testing"
	"time"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProbeThrottledUsesPersistedNextAllowedAt(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	state := &providerv1.ProviderProbeRunState{
		NextAllowedAt: timestamppb.New(now.Add(time.Minute)),
	}

	if !probeThrottled(state, now) {
		t.Fatal("probeThrottled() = false, want true before next_allowed_at")
	}
	if probeThrottled(state, now.Add(2*time.Minute)) {
		t.Fatal("probeThrottled() = true, want false after next_allowed_at")
	}
}

func TestNormalizeProbeTriggerDefaultsToManual(t *testing.T) {
	got := normalizeProbeTrigger(providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_UNSPECIFIED)
	if got != providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_MANUAL {
		t.Fatalf("normalizeProbeTrigger() = %v, want MANUAL", got)
	}
}
