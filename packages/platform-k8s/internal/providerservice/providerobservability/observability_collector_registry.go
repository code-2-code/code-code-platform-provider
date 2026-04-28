package providerobservability

import (
	"sort"
	"strings"
)

type observabilityCollectorFactory func() ObservabilityCollector

var (
	vendorCollectorFactories = map[string]observabilityCollectorFactory{}
	oauthCollectorFactories  = map[string]observabilityCollectorFactory{}
)

func registerVendorCollectorFactory(collectorID string, factory observabilityCollectorFactory) {
	trimmedCollectorID := strings.TrimSpace(collectorID)
	if trimmedCollectorID == "" || factory == nil {
		return
	}
	vendorCollectorFactories[trimmedCollectorID] = factory
}

func registerOAuthCollectorFactory(collectorID string, factory observabilityCollectorFactory) {
	trimmedCollectorID := strings.TrimSpace(collectorID)
	if trimmedCollectorID == "" || factory == nil {
		return
	}
	oauthCollectorFactories[trimmedCollectorID] = factory
}

func DefaultVendorCollectors() []ObservabilityCollector {
	return buildCollectors(vendorCollectorFactories)
}

func DefaultOAuthCollectors() []ObservabilityCollector {
	return buildCollectors(oauthCollectorFactories)
}

func buildCollectors(factories map[string]observabilityCollectorFactory) []ObservabilityCollector {
	collectorIDs := make([]string, 0, len(factories))
	for collectorID := range factories {
		collectorIDs = append(collectorIDs, collectorID)
	}
	sort.Strings(collectorIDs)
	collectors := make([]ObservabilityCollector, 0, len(collectorIDs))
	for _, collectorID := range collectorIDs {
		factory := factories[collectorID]
		if factory == nil {
			continue
		}
		if collector := factory(); collector != nil {
			collectors = append(collectors, collector)
		}
	}
	return collectors
}
