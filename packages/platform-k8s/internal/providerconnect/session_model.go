package providerconnect

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
)

type sessionRecord struct {
	SessionID      string `json:"sessionId"`
	OAuthSessionID string `json:"oauthSessionId"`
	sessionTargetSnapshot
	sessionProgress
}

func newSessionRecord(sessionID string, target *connectTarget, status *credentialv1.OAuthAuthorizationSessionStatus) (*sessionRecord, error) {
	snapshot, err := newSessionTargetSnapshot(target)
	if err != nil {
		return nil, err
	}
	record := &sessionRecord{
		SessionID:             targetDisplayValue(sessionID),
		OAuthSessionID:        targetDisplayValue(sessionID),
		sessionTargetSnapshot: snapshot,
	}
	record.applyOAuthStatus(status)
	return record, nil
}

func (r *sessionRecord) needsFinalize() bool {
	return r != nil && r.sessionTargetSnapshot.needsFinalize(r.ConnectedSurfaceID)
}

func (r *sessionRecord) target() (*connectTarget, error) {
	models, err := r.sessionTargetSnapshot.models()
	if err != nil {
		return nil, err
	}
	return r.sessionTargetSnapshot.target(models), nil
}

func (r *sessionRecord) view(provider *ProviderView, oauthState *credentialv1.OAuthAuthorizationSessionState) *SessionView {
	view := &SessionView{
		SessionID:        r.SessionID,
		OAuthSessionID:   r.OAuthSessionID,
		Phase:            r.Phase,
		DisplayName:      r.DisplayName,
		AuthorizationURL: r.AuthorizationURL,
		UserCode:         r.UserCode,
		Message:          r.Message,
		ErrorMessage:     r.ErrorMessage,
		AddMethod:        r.AddMethod,
		CLIID:            r.CLIID,
		Provider:         provider,
	}
	if view.AuthorizationURL == "" && oauthState != nil {
		view.AuthorizationURL = oauthState.GetStatus().GetAuthorizationUrl()
	}
	if view.UserCode == "" && oauthState != nil {
		view.UserCode = oauthState.GetStatus().GetUserCode()
	}
	return view
}

func targetDisplayValue(value string) string {
	return strings.TrimSpace(value)
}
