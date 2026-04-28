package provideridentity

import "strings"

// ObservabilityCredentialID returns the stable management-plane credential id
// derived from one provider id.
func ObservabilityCredentialID(providerID string) string {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ""
	}
	return providerID + "-observability"
}
