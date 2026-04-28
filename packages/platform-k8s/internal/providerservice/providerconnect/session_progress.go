package providerconnect

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
)

type sessionProgress struct {
	Phase              SessionPhase `json:"phase"`
	AuthorizationURL   string       `json:"authorizationUrl"`
	UserCode           string       `json:"userCode"`
	Message            string       `json:"message"`
	ErrorMessage       string       `json:"errorMessage"`
	ConnectedSurfaceID string       `json:"connectedSurfaceId"`
}

func (p *sessionProgress) terminal() bool {
	switch p.Phase {
	case SessionPhaseSucceeded,
		SessionPhaseFailed,
		SessionPhaseExpired,
		SessionPhaseCanceled:
		return true
	default:
		return false
	}
}

func (p *sessionProgress) applyOAuthReadError(err error) {
	p.Phase = SessionPhaseFailed
	p.ErrorMessage = err.Error()
	p.Message = "Provider connect session could not read OAuth state."
}

func (p *sessionProgress) applyOAuthStatus(status *credentialv1.OAuthAuthorizationSessionStatus) {
	phase, message, errorMessage := connectPhaseFromOAuthStatus(status)
	p.Phase = phase
	p.Message = message
	p.ErrorMessage = errorMessage
	if status == nil {
		p.AuthorizationURL = ""
		p.UserCode = ""
		return
	}
	p.AuthorizationURL = strings.TrimSpace(status.GetAuthorizationUrl())
	p.UserCode = strings.TrimSpace(status.GetUserCode())
}

func (p *sessionProgress) succeeded() bool {
	return p != nil && p.Phase == SessionPhaseSucceeded
}

func (p *sessionProgress) markAuthenticationUpdated() {
	p.Message = "Provider authentication updated."
	p.ErrorMessage = ""
}

func (p *sessionProgress) markFinalizeFailed(err error) {
	p.Phase = SessionPhaseFailed
	p.ErrorMessage = err.Error()
	p.Message = "Provider surface binding creation failed."
}

func (p *sessionProgress) markConnected(surfaceID string) {
	p.Phase = SessionPhaseSucceeded
	p.ConnectedSurfaceID = strings.TrimSpace(surfaceID)
	p.Message = "Provider connected."
	p.ErrorMessage = ""
}

func connectPhaseFromOAuthStatus(status *credentialv1.OAuthAuthorizationSessionStatus) (SessionPhase, string, string) {
	if status == nil {
		return SessionPhasePending, "Waiting for authorization session state.", ""
	}
	message := strings.TrimSpace(status.GetMessage())
	switch status.GetPhase() {
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PENDING:
		return SessionPhasePending, message, ""
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_AWAITING_USER:
		return SessionPhaseAwaitingUser, message, ""
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PROCESSING:
		return SessionPhaseProcessing, message, ""
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED:
		return SessionPhaseSucceeded, message, ""
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_EXPIRED:
		return SessionPhaseExpired, message, message
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_CANCELED:
		return SessionPhaseCanceled, message, message
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_FAILED:
		return SessionPhaseFailed, message, message
	default:
		return SessionPhasePending, message, ""
	}
}
