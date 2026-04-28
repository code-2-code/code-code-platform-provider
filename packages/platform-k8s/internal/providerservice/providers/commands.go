package providers

// UpdateProviderCommand carries one provider rename request.
type UpdateProviderCommand struct {
	DisplayName string
}

// UpdateAPIKeyAuthenticationCommand carries one API key authentication update.
type UpdateAPIKeyAuthenticationCommand struct {
	APIKey string
}

// UpdateObservabilityAuthenticationCommand carries one management-plane auth update.
type UpdateObservabilityAuthenticationCommand struct {
	SchemaID     string
	RequiredKeys []string
	Values       map[string]string
}
