package providerobservability

import (
	"strings"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const observabilityProbeReasonEnumPrefix = "PROVIDER_OBSERVABILITY_PROBE_REASON_"

func observabilityProbeReasonLabel(reason providerservicev1.ProviderObservabilityProbeReason) string {
	if reason == providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UNSPECIFIED {
		return ""
	}
	name := reason.String()
	if !strings.HasPrefix(name, observabilityProbeReasonEnumPrefix) {
		return ""
	}
	return strings.TrimPrefix(name, observabilityProbeReasonEnumPrefix)
}

func observabilityProbeReasonFromLabel(label string) (providerservicev1.ProviderObservabilityProbeReason, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(label))
	if normalized == "" {
		return providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UNSPECIFIED, false
	}
	value := providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UNSPECIFIED.
		Descriptor().
		Values().
		ByName(protoreflect.Name(observabilityProbeReasonEnumPrefix + normalized))
	if value == nil || value.Number() == 0 {
		return providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UNSPECIFIED, false
	}
	return providerservicev1.ProviderObservabilityProbeReason(value.Number()), true
}

func observabilityProbeReasonLabels() []string {
	values := providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UNSPECIFIED.
		Descriptor().
		Values()
	labels := make([]string, 0, values.Len()-1)
	for i := 0; i < values.Len(); i++ {
		value := values.Get(i)
		if value.Number() == 0 {
			continue
		}
		label := observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason(value.Number()))
		if label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

var (
	observabilityReasonPlatformUnavailable = observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_PLATFORM_UNAVAILABLE)
	observabilityReasonUpstreamUnreachable = observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UPSTREAM_UNREACHABLE)
	observabilityReasonUpstreamTimeout     = observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_UPSTREAM_TIMEOUT)
	observabilityReasonAuthBlocked         = observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_AUTH_BLOCKED)
	observabilityReasonCredentialsMissing  = observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_CREDENTIALS_MISSING)
	observabilityReasonProbeFailed         = observabilityProbeReasonLabel(providerservicev1.ProviderObservabilityProbeReason_PROVIDER_OBSERVABILITY_PROBE_REASON_PROBE_FAILED)
)
