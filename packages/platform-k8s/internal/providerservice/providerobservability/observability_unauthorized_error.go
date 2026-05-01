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

func observabilityUnauthorizedSafeMessage(err error) string {
	var target *observabilityUnauthorizedError
	if !errors.As(err, &target) {
		return "observability credential unauthorized"
	}
	message := strings.Join(strings.Fields(strings.TrimSpace(target.message)), " ")
	if message == "" {
		return "observability credential unauthorized"
	}
	lower := strings.ToLower(message)
	for _, statusCode := range []string{"status 401", "status 403"} {
		if idx := strings.Index(lower, statusCode); idx >= 0 {
			return trimObservabilityUnauthorizedMessage(message[:idx+len(statusCode)])
		}
	}
	if strings.Contains(lower, "egress auth replacement failed") ||
		strings.Contains(lower, "request auth headers") ||
		strings.Contains(lower, "observability credential") ||
		strings.Contains(lower, "credential is required") {
		return trimObservabilityUnauthorizedMessage(message)
	}
	return "observability credential unauthorized"
}

func trimObservabilityUnauthorizedMessage(message string) string {
	message = strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	const maxLen = 240
	if len(message) <= maxLen {
		return message
	}
	return strings.TrimSpace(message[:maxLen]) + "..."
}

func observabilityAuthBlockedReason(message string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if normalized == "" {
		return ""
	}
	for _, token := range strings.FieldsFunc(normalized, func(r rune) bool {
		return !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_')
	}) {
		if token == observabilityReasonCredentialsMissing {
			return observabilityReasonCredentialsMissing
		}
	}
	lower := strings.ToLower(normalized)
	switch {
	case strings.Contains(lower, "credential") && strings.Contains(lower, "missing"):
		return observabilityReasonCredentialsMissing
	case strings.Contains(lower, "status 401"):
		return observabilityReasonAuthBlocked
	case strings.Contains(lower, "status 403"):
		return observabilityReasonAuthBlocked
	default:
		return ""
	}
}
