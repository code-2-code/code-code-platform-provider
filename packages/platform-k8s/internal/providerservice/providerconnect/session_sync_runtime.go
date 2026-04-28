package providerconnect

import (
	"context"
	"fmt"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
)

type providerConnectSessionSyncRuntime struct {
	oauth     oauthSessionService
	finalizer providerConnectSessionFinalizer
}

func newProviderConnectSessionSyncRuntime(
	oauth oauthSessionService,
	finalizer providerConnectSessionFinalizer,
) providerConnectSessionSyncRuntime {
	return providerConnectSessionSyncRuntime{
		oauth:     oauth,
		finalizer: finalizer,
	}
}

func (r providerConnectSessionSyncRuntime) Sync(
	ctx context.Context,
	record *sessionRecord,
) (*sessionRecord, *credentialv1.OAuthAuthorizationSessionState, error) {
	if record == nil {
		return nil, nil, domainerror.NewValidation("platformk8s/providerconnect: session record is nil")
	}
	if r.oauth == nil {
		return nil, nil, fmt.Errorf("platformk8s/providerconnect: oauth session runtime is incomplete")
	}
	oauthState, err := r.oauth.GetSession(ctx, record.OAuthSessionID)
	if err != nil {
		if record.terminal() {
			return record, nil, nil
		}
		record.applyOAuthReadError(err)
		return record, nil, nil
	}
	record.applyOAuthStatus(oauthState.GetStatus())
	if record.needsFinalize() && oauthStateAllowsProviderMaterialization(oauthState) {
		if r.finalizer == nil {
			return nil, nil, fmt.Errorf("platformk8s/providerconnect: oauth session finalizer is nil")
		}
		surface, err := r.finalizer.Finalize(ctx, record, oauthState)
		if err != nil {
			record.markFinalizeFailed(err)
			return record, oauthState, nil
		}
		record.markConnected(surface.GetSurfaceId())
		return record, oauthState, nil
	}
	if record.succeeded() && !record.needsFinalize() {
		record.markAuthenticationUpdated()
		return record, oauthState, nil
	}
	return record, oauthState, nil
}

func oauthStateAllowsProviderMaterialization(state *credentialv1.OAuthAuthorizationSessionState) bool {
	switch state.GetStatus().GetPhase() {
	case credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED:
		return strings.TrimSpace(state.GetSpec().GetTargetCredentialId()) != "" ||
			strings.TrimSpace(state.GetStatus().GetImportedCredential().GetCredentialId()) != ""
	default:
		return false
	}
}
