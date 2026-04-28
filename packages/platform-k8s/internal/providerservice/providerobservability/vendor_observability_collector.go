package providerobservability

import (
	"context"
	"errors"
	"net/http"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
)

const (
	vendorObservabilityMaxBodyReadSize = 1 << 16
)

// VendorObservabilityCollector probes one vendor API-key management surface.
type VendorObservabilityCollector interface {
	CollectorID() string
	Collect(context.Context, VendorObservabilityCollectInput) (*VendorObservabilityCollectResult, error)
}

type VendorObservabilityAuthAdapter interface {
	AuthAdapterID() string
}

// VendorObservabilityCollectInput carries one collector execution context.
type VendorObservabilityCollectInput struct {
	VendorID                 string
	ProviderID               string
	ProviderSurfaceBindingID string
	CredentialID             string
	SurfaceBaseURL           string
	APIKey                   string
	// ObservabilityCredential carries one optional management-plane override
	// resolved from vendor support metadata or account-owned credential.
	// Collectors inspect its kind instead of assuming one token-shaped field.
	ObservabilityCredential *credentialv1.ResolvedCredential
	CredentialBackfills     []CredentialBackfillRule
	HTTPClient              *http.Client
}

// VendorObservabilityCollectResult carries active operation metric values from one
// collector execution.
type VendorObservabilityCollectResult struct {
	GaugeRows                []VendorObservabilityMetricRow
	CredentialBackfillValues map[string]string
}

type VendorObservabilityMetricRow struct {
	MetricName string
	Labels     map[string]string
	Value      float64
}

type vendorObservabilityUnauthorizedError struct {
	message string
	reason  string
}

func (e *vendorObservabilityUnauthorizedError) Error() string {
	return e.message
}

func unauthorizedVendorObservabilityError(message string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		trimmed = "vendor observability unauthorized"
	}
	return &vendorObservabilityUnauthorizedError{
		message: trimmed,
		reason:  vendorObservabilityAuthBlockedReason(trimmed),
	}
}

func isVendorObservabilityUnauthorizedError(err error) bool {
	var target *vendorObservabilityUnauthorizedError
	return errors.As(err, &target)
}

func vendorObservabilityUnauthorizedReason(err error) string {
	var target *vendorObservabilityUnauthorizedError
	if errors.As(err, &target) {
		return strings.TrimSpace(target.reason)
	}
	return ""
}

func vendorObservabilityAuthBlockedReason(message string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if normalized == "" {
		return ""
	}
	for _, token := range strings.FieldsFunc(normalized, func(r rune) bool {
		return !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_')
	}) {
		if len(token) >= 3 && strings.Contains(token, "_") && token == strings.ToUpper(token) {
			return token
		}
	}
	lower := strings.ToLower(normalized)
	switch {
	case strings.Contains(lower, "status 401"):
		return "HTTP_401_UNAUTHORIZED"
	case strings.Contains(lower, "status 403"):
		return "HTTP_403_FORBIDDEN"
	default:
		return ""
	}
}

func observabilityCredentialToken(resolved *credentialv1.ResolvedCredential) string {
	if resolved == nil {
		return ""
	}
	switch resolved.GetKind() {
	case credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY:
		return strings.TrimSpace(resolved.GetApiKey().GetApiKey())
	case credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH:
		return strings.TrimSpace(resolved.GetOauth().GetAccessToken())
	case credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION:
		return observabilitySessionValue(resolved, "token", "access_token", "api_key", "session_token", "authjs_session_token", "authjs.session-token")
	default:
		return ""
	}
}

func observabilitySessionValue(resolved *credentialv1.ResolvedCredential, keys ...string) string {
	if resolved == nil || resolved.GetKind() != credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION || resolved.GetSession() == nil {
		return ""
	}
	values := resolved.GetSession().GetValues()
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	return ""
}
