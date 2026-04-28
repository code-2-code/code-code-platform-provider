package providers

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/provideridentity"
)

func (p *ProviderProjection) ObservabilityCredentialID() string {
	return provideridentity.ObservabilityCredentialID(p.ID())
}

func (p *ProviderProjection) SubjectSummaryCredentialID() string {
	if p.AuthKind() == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
		if credentialID := p.ObservabilityCredentialID(); strings.TrimSpace(credentialID) != "" {
			return credentialID
		}
	}
	return p.CredentialID()
}

func (p *ProviderProjection) ObservabilityCredential(command UpdateObservabilityAuthenticationCommand) (*CredentialSessionUpdate, error) {
	if err := p.ValidateMutable(); err != nil {
		return nil, err
	}
	if p.AuthKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
		return nil, domainerror.NewValidation("platformk8s/providers: provider %q does not accept observability authentication update", p.ID())
	}
	values := trimObservabilityValues(command.Values)
	if len(values) == 0 {
		return nil, nil
	}
	schemaID := strings.TrimSpace(command.SchemaID)
	if schemaID == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: observability session schema_id is required")
	}
	return &CredentialSessionUpdate{
		CredentialID: p.ObservabilityCredentialID(),
		DisplayName:  p.observabilityCredentialDisplayName(p.DisplayName()),
		Purpose:      credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_MANAGEMENT_PLANE,
		VendorID:     p.VendorID(),
		SchemaID:     schemaID,
		RequiredKeys: trimObservabilityKeys(command.RequiredKeys),
		Values:       values,
		MergeValues:  true,
	}, nil
}

func (p *ProviderProjection) observabilityCredentialDisplayName(displayName string) string {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = p.DisplayName()
	}
	return displayName + " Observability"
}

func trimObservabilityValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		trimmed[key] = value
	}
	if len(trimmed) == 0 {
		return nil
	}
	return trimmed
}

func trimObservabilityKeys(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) == 0 {
		return nil
	}
	return trimmed
}
