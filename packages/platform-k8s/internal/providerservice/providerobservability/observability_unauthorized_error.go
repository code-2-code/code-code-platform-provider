package providerobservability

import (
	"errors"
	"strings"
)

// observabilityUnauthorizedError signals an auth failure during observability
// collection. Both vendor and OAuth collectors use this single error type.
type observabilityUnauthorizedError struct {
	message string
	reason  string
}

func (e *observabilityUnauthorizedError) Error() string {
	return e.message
}

// unauthorizedObservabilityError creates an auth-blocked error with automatic
// machine-readable reason extraction from the message.
func unauthorizedObservabilityError(message string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		trimmed = "observability unauthorized"
	}
	return &observabilityUnauthorizedError{
		message: trimmed,
		reason:  observabilityAuthBlockedReason(trimmed),
	}
}

func isObservabilityUnauthorizedError(err error) bool {
	var target *observabilityUnauthorizedError
	return errors.As(err, &target)
}

func observabilityUnauthorizedReason(err error) string {
	var target *observabilityUnauthorizedError
	if errors.As(err, &target) {
		return strings.TrimSpace(target.reason)
	}
	return ""
}

func observabilityAuthBlockedReason(message string) string {
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
