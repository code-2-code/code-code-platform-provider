package providerobservability

import (
	"strings"
	"sync"
	"time"
)

// probeStateTracker manages throttle state and metric recording for one
// observability runner. It is embedded by both VendorObservabilityRunner and
// OAuthObservabilityRunner to eliminate duplicated state management.
type probeStateTracker struct {
	metrics        *observabilityMetrics
	defaultBackoff time.Duration

	mu     sync.Mutex
	states map[string]probeState
}

type probeState struct {
	lastAttemptAt time.Time
	nextAllowedAt time.Time
}

func newProbeStateTracker(metrics *observabilityMetrics, defaultBackoff time.Duration) probeStateTracker {
	return probeStateTracker{
		metrics:        metrics,
		defaultBackoff: defaultBackoff,
		states:         map[string]probeState{},
	}
}

func (t *probeStateTracker) throttled(providerID string, providerSurfaceBindingID string, now time.Time) (bool, time.Time) {
	stateKey := probeStateKey(providerID, providerSurfaceBindingID)
	if stateKey == "" {
		return false, time.Time{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	state, ok := t.states[stateKey]
	if !ok || state.nextAllowedAt.IsZero() {
		return false, time.Time{}
	}
	if now.Before(state.nextAllowedAt) {
		return true, state.nextAllowedAt
	}
	return false, state.nextAllowedAt
}

func (t *probeStateTracker) nextAllowed(providerID string, providerSurfaceBindingID string) time.Time {
	stateKey := probeStateKey(providerID, providerSurfaceBindingID)
	if stateKey == "" {
		return time.Time{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	state, ok := t.states[stateKey]
	if !ok {
		return time.Time{}
	}
	return state.nextAllowedAt
}

// recordState persists the throttle state and records metrics for a probe result.
// It returns the mutated result with timing fields set.
func (t *probeStateTracker) recordState(
	result *ProbeResult,
	trigger Trigger,
	now time.Time,
	backoff time.Duration,
) *ProbeResult {
	if result == nil {
		return nil
	}
	if backoff <= 0 {
		backoff = t.defaultBackoff
	}
	nextAllowedAt := now.Add(backoff)
	stateKey := probeStateKey(result.ProviderID, result.ProviderSurfaceBindingID)
	if stateKey == "" {
		stateKey = probeStateKey("", result.ProviderSurfaceBindingID)
	}
	t.mu.Lock()
	t.states[stateKey] = probeState{
		lastAttemptAt: now,
		nextAllowedAt: nextAllowedAt,
	}
	t.mu.Unlock()
	result.LastAttemptAt = timePointerCopy(&now)
	result.NextAllowedAt = timePointerCopy(&nextAllowedAt)
	t.metrics.record(result.OwnerID, result.ProviderID, trigger, result.Outcome, result.Reason, now, nextAllowedAt)
	return result
}

func probeStateKey(providerID string, providerSurfaceBindingID string) string {
	if providerID := strings.TrimSpace(providerID); providerID != "" {
		return "provider:" + providerID
	}
	if surfaceID := strings.TrimSpace(providerSurfaceBindingID); surfaceID != "" {
		return "surface:" + surfaceID
	}
	return ""
}
