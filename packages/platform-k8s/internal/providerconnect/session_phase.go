package providerconnect

// SessionPhase is the providerconnect-owned connect-session phase.
type SessionPhase string

const (
	SessionPhaseUnspecified  SessionPhase = ""
	SessionPhasePending      SessionPhase = "pending"
	SessionPhaseAwaitingUser SessionPhase = "awaiting_user"
	SessionPhaseProcessing   SessionPhase = "processing"
	SessionPhaseSucceeded    SessionPhase = "succeeded"
	SessionPhaseFailed       SessionPhase = "failed"
	SessionPhaseExpired      SessionPhase = "expired"
	SessionPhaseCanceled     SessionPhase = "canceled"
)
