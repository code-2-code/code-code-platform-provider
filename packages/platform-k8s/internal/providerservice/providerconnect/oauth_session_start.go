package providerconnect

import (
	"context"
	"fmt"

	credentialv1 "code-code.internal/go-contract/credential/v1"
)

type providerConnectSessionStartRuntime struct {
	sessions providerConnectSessions
	views    providerConnectSessionViewRuntime
}

func newProviderConnectSessionStartRuntime(
	sessions providerConnectSessions,
	views providerConnectSessionViewRuntime,
) providerConnectSessionStartRuntime {
	return providerConnectSessionStartRuntime{
		sessions: sessions,
		views:    views,
	}
}

type oauthSessionStartExecution struct {
	target *connectTarget
	flow   credentialv1.OAuthAuthorizationFlow
}

func newOAuthSessionStartExecution(
	target *connectTarget,
	flow credentialv1.OAuthAuthorizationFlow,
) *oauthSessionStartExecution {
	return &oauthSessionStartExecution{
		target: target,
		flow:   flow,
	}
}

func (e *oauthSessionStartExecution) Execute(
	ctx context.Context,
	runtime providerConnectSessionStartRuntime,
) (*SessionView, error) {
	if e == nil || e.target == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: oauth session target is nil")
	}
	if runtime.sessions.oauth == nil || runtime.sessions.store == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: oauth session runtime is incomplete")
	}
	session, err := runtime.sessions.oauth.StartSession(ctx, e.target.OAuthSessionSpec(e.flow))
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: start oauth session: %w", err)
	}
	record, err := newSessionRecord(session.GetSpec().GetSessionId(), e.target, session.GetStatus())
	if err != nil {
		return nil, err
	}
	if err := runtime.sessions.store.create(ctx, record); err != nil {
		_, _ = runtime.sessions.oauth.CancelSession(context.Background(), session.GetSpec().GetSessionId())
		return nil, err
	}
	return runtime.views.View(ctx, record, session)
}
