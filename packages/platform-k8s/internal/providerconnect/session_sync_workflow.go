package providerconnect

import (
	"context"
	"strings"
)

// SessionSyncWorkflowRuntime submits durable provider-connect session sync work.
type SessionSyncWorkflowRuntime interface {
	SubmitSessionSync(ctx context.Context, sessionID string) error
}

type providerConnectSessionSyncWorkflow struct {
	runtime SessionSyncWorkflowRuntime
}

func newProviderConnectSessionSyncWorkflow(runtime SessionSyncWorkflowRuntime) *providerConnectSessionSyncWorkflow {
	return &providerConnectSessionSyncWorkflow{runtime: runtime}
}

func (w *providerConnectSessionSyncWorkflow) Trigger(ctx context.Context, sessionID string) error {
	if w == nil || w.runtime == nil {
		return nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	return w.runtime.SubmitSessionSync(ctx, sessionID)
}
