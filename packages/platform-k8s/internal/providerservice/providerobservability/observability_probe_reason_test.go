package providerobservability

import (
	"testing"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

func TestObservabilityProbeReasonLabelsComeFromProto(t *testing.T) {
	t.Parallel()

	labels := map[string]bool{}
	for _, label := range observabilityProbeReasonLabels() {
		labels[label] = true
	}
	for _, want := range []string{
		"UPSTREAM_UNREACHABLE",
		"CREDENTIALS_MISSING",
		"AUTH_BLOCKED",
	} {
		if !labels[want] {
			t.Fatalf("labels missing %q", want)
		}
	}

	got, ok := observabilityProbeReasonFromLabel("credentials_missing")
	if !ok {
		t.Fatal("credentials_missing did not resolve")
	}
	if want := providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_CREDENTIALS_MISSING; got != want {
		t.Fatalf("reason = %v, want %v", got, want)
	}

	if _, ok := observabilityProbeReasonFromLabel("UNKNOWN_VENDOR_REASON"); ok {
		t.Fatal("unknown reason unexpectedly resolved")
	}
}

func TestObservabilityAuthBlockedReasonUsesKnownProtoReasons(t *testing.T) {
	t.Parallel()

	if got, want := observabilityAuthBlockedReason("provider returned CREDENTIALS_MISSING"), "CREDENTIALS_MISSING"; got != want {
		t.Fatalf("known token reason = %q, want %q", got, want)
	}
	if got := observabilityAuthBlockedReason("provider returned UNKNOWN_VENDOR_REASON"); got != "" {
		t.Fatalf("unknown token reason = %q, want empty", got)
	}
	if got, want := observabilityAuthBlockedReason("provider returned status 401"), "AUTH_BLOCKED"; got != want {
		t.Fatalf("status reason = %q, want %q", got, want)
	}
}
