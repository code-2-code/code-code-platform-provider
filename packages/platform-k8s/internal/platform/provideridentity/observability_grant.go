package provideridentity

import "strings"

// ObservabilityGrantID returns the stable management-plane grant id derived from
// one provider id.
func ObservabilityGrantID(providerID string) string {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ""
	}
	return providerID + "-observability"
}
