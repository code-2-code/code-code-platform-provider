package providerobservability

import (
	"sort"
	"strings"
)

type vendorObservabilityCollectorFactory func() VendorObservabilityCollector

var vendorObservabilityCollectorFactories = map[string]vendorObservabilityCollectorFactory{}

func registerVendorObservabilityCollectorFactory(collectorID string, factory vendorObservabilityCollectorFactory) {
	trimmedCollectorID := strings.TrimSpace(collectorID)
	if trimmedCollectorID == "" || factory == nil {
		return
	}
	vendorObservabilityCollectorFactories[trimmedCollectorID] = factory
}

func DefaultVendorObservabilityCollectors() []VendorObservabilityCollector {
	collectorIDs := make([]string, 0, len(vendorObservabilityCollectorFactories))
	for collectorID := range vendorObservabilityCollectorFactories {
		collectorIDs = append(collectorIDs, collectorID)
	}
	sort.Strings(collectorIDs)
	collectors := make([]VendorObservabilityCollector, 0, len(collectorIDs))
	for _, collectorID := range collectorIDs {
		factory := vendorObservabilityCollectorFactories[collectorID]
		if factory == nil {
			continue
		}
		if collector := factory(); collector != nil {
			collectors = append(collectors, collector)
		}
	}
	return collectors
}
