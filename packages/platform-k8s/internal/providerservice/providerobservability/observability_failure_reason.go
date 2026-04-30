package providerobservability

import (
	"context"
	"strings"
	"time"
)

func observabilityFailureReasonFromError(err error) string {
	if err == nil {
		return ""
	}
	return observabilityFailureReason(err.Error())
}

func observabilityFailureReason(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return ""
	}
	isKubernetesAPI := strings.Contains(normalized, "10.96.0.1:443") ||
		strings.Contains(normalized, "kubernetes.default") ||
		strings.Contains(normalized, "platformk8s:")
	if isKubernetesAPI && hasAnyObservabilityErrorToken(
		normalized,
		"connect: connection refused",
		"tls handshake timeout",
		"context deadline exceeded",
		"client.timeout exceeded",
		"i/o timeout",
	) {
		return observabilityReasonPlatformUnavailable
	}
	switch {
	case strings.Contains(normalized, "connect: connection refused"):
		return observabilityReasonUpstreamUnreachable
	case strings.Contains(normalized, "tls handshake timeout"):
		return observabilityReasonUpstreamUnreachable
	case strings.Contains(normalized, "context deadline exceeded") ||
		strings.Contains(normalized, "client.timeout exceeded") ||
		strings.Contains(normalized, "i/o timeout"):
		return observabilityReasonUpstreamTimeout
	case strings.Contains(normalized, "no such host"):
		return observabilityReasonUpstreamUnreachable
	case strings.Contains(normalized, "connection reset by peer"):
		return observabilityReasonUpstreamUnreachable
	default:
		return observabilityReasonProbeFailed
	}
}

func observabilityTransientPlatformError(err error) bool {
	return observabilityFailureReasonFromError(err) == observabilityReasonPlatformUnavailable
}

func retryObservabilityTransientPlatform(ctx context.Context, fn func() error) error {
	const attempts = 3
	for attempt := 0; attempt < attempts; attempt++ {
		err := fn()
		if err == nil || !observabilityTransientPlatformError(err) || attempt == attempts-1 {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 250 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func hasAnyObservabilityErrorToken(value string, tokens ...string) bool {
	for _, token := range tokens {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}
