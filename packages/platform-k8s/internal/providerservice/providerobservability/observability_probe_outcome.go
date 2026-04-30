package providerobservability

import "time"

// probeFailure sets a generic failure outcome on the result and records it
// via the probeStateTracker. Used by both vendor and OAuth probe paths
// to eliminate the repeated "set outcome → set reason → set message →
// recordState → return" boilerplate.
func probeFailure(
	tracker *probeStateTracker,
	result *ProbeResult,
	err error,
	trigger Trigger,
	now time.Time,
	backoff time.Duration,
) *ProbeResult {
	result.Outcome = ProbeOutcomeFailed
	result.Reason = observabilityFailureReasonFromError(err)
	result.Message = err.Error()
	return tracker.recordState(result, trigger, now, backoff)
}

// probeUnsupported records an unsupported outcome with a message.
// The outcome stays at ProbeOutcomeUnsupported (the default).
func probeUnsupported(
	tracker *probeStateTracker,
	result *ProbeResult,
	message string,
	trigger Trigger,
	now time.Time,
	backoff time.Duration,
) *ProbeResult {
	result.Message = message
	return tracker.recordState(result, trigger, now, backoff)
}

// probeCollectFailure handles the error returned by collector.Collect().
// It distinguishes unauthorized errors (→ AuthBlocked) from generic failures.
// The reasonFn allows vendor and OAuth to use different reason extraction logic.
func probeCollectFailure(
	tracker *probeStateTracker,
	result *ProbeResult,
	collectErr error,
	trigger Trigger,
	now time.Time,
	backoff time.Duration,
	reasonFn func(ProbeOutcome, string) string,
) *ProbeResult {
	if isObservabilityUnauthorizedError(collectErr) {
		result.Outcome = ProbeOutcomeAuthBlocked
	} else {
		result.Outcome = ProbeOutcomeFailed
	}
	if reasonFn != nil {
		result.Reason = reasonFn(result.Outcome, collectErr.Error())
	} else {
		if result.Outcome == ProbeOutcomeAuthBlocked {
			result.Reason = observabilityUnauthorizedReason(collectErr)
		} else {
			result.Reason = observabilityFailureReasonFromError(collectErr)
		}
	}
	result.Message = collectErr.Error()
	return tracker.recordState(result, trigger, now, backoff)
}

// probeThrottled checks whether the probe is throttled and, if so, records
// the throttled outcome. Returns (result, true) if throttled; (nil, false) otherwise.
func probeThrottled(
	tracker *probeStateTracker,
	metrics *observabilityMetrics,
	result *ProbeResult,
	ownerID string,
	trigger Trigger,
	now time.Time,
) (*ProbeResult, bool) {
	if trigger == TriggerManual {
		return nil, false
	}
	throttled, nextAllowedAt := tracker.throttled(result.ProviderID, result.SurfaceID, now)
	if !throttled {
		return nil, false
	}
	result.Outcome = ProbeOutcomeThrottled
	result.Message = "operation is throttled by minimum interval"
	result.LastAttemptAt = timePointerCopy(&now)
	result.NextAllowedAt = timePointerCopy(&nextAllowedAt)
	metrics.recordThrottle(ownerID, result.ProviderID, trigger)
	return result, true
}
