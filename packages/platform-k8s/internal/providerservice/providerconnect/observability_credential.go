package providerconnect

import (
	"strings"

	"code-code.internal/platform-k8s/internal/platform/provideridentity"
)

func (t *connectTarget) ObservabilityCredential(token string) *CredentialSessionCreate {
	if t == nil {
		return nil
	}
	return newObservabilityCredential(t.TargetProviderID, t.DisplayName, t.VendorID, token)
}

func (p *connectPlan) ObservabilityCredential(token string) *CredentialSessionCreate {
	if p == nil {
		return nil
	}
	return newObservabilityCredential(p.TargetProviderID, p.DisplayName, p.VendorID, token)
}

func newObservabilityCredential(providerID, displayName, vendorID, token string) *CredentialSessionCreate {
	token = strings.TrimSpace(token)
	providerID = strings.TrimSpace(providerID)
	if token == "" || providerID == "" {
		return nil
	}
	schemaID, requiredKey, ok := observabilitySchema(vendorID)
	if !ok {
		return nil
	}
	return &CredentialSessionCreate{
		CredentialID: provideridentity.ObservabilityCredentialID(providerID),
		DisplayName:  strings.TrimSpace(displayName) + " Observability",
		VendorID:     strings.TrimSpace(vendorID),
		SchemaID:     schemaID,
		RequiredKeys: []string{requiredKey},
		Values: map[string]string{
			requiredKey: token,
		},
	}
}

func observabilitySchema(vendorID string) (string, string, bool) {
	switch strings.TrimSpace(vendorID) {
	case "cerebras":
		return "cerebras-session", "authjs_session_token", true
	default:
		return "", "", false
	}
}
