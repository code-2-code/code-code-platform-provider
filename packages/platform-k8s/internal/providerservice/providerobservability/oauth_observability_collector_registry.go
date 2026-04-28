package providerobservability

import (
	"sort"
	"strings"
)

type oauthObservabilityCollectorFactory func() OAuthObservabilityCollector

var oauthObservabilityCollectorFactories = map[string]oauthObservabilityCollectorFactory{}

func registerOAuthObservabilityCollectorFactory(collectorID string, factory oauthObservabilityCollectorFactory) {
	trimmedCollectorID := strings.TrimSpace(collectorID)
	if trimmedCollectorID == "" || factory == nil {
		return
	}
	oauthObservabilityCollectorFactories[trimmedCollectorID] = factory
}

func DefaultOAuthObservabilityCollectors() []OAuthObservabilityCollector {
	collectorIDs := make([]string, 0, len(oauthObservabilityCollectorFactories))
	for collectorID := range oauthObservabilityCollectorFactories {
		collectorIDs = append(collectorIDs, collectorID)
	}
	sort.Strings(collectorIDs)
	collectors := make([]OAuthObservabilityCollector, 0, len(collectorIDs))
	for _, collectorID := range collectorIDs {
		factory := oauthObservabilityCollectorFactories[collectorID]
		if factory == nil {
			continue
		}
		if collector := factory(); collector != nil {
			collectors = append(collectors, collector)
		}
	}
	return collectors
}
