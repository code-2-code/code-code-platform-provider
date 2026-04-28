package providerconnect

import (
	"context"
	"log/slog"
)

// PostConnectWorkflowRuntime submits durable provider-connect post-connect workflows.
type PostConnectWorkflowRuntime interface {
	SubmitPostConnect(ctx context.Context, providerID string) error
}

type providerConnectPostConnectWorkflow struct {
	runtime PostConnectWorkflowRuntime
	logger  *slog.Logger
}

func newProviderConnectPostConnectWorkflow(
	runtime PostConnectWorkflowRuntime,
	logger *slog.Logger,
) *providerConnectPostConnectWorkflow {
	if logger == nil {
		logger = slog.Default()
	}
	return &providerConnectPostConnectWorkflow{
		runtime: runtime,
		logger:  logger,
	}
}

func (w *providerConnectPostConnectWorkflow) Dispatch(ctx context.Context, providerID string) {
	if w == nil || w.runtime == nil {
		return
	}
	if err := w.runtime.SubmitPostConnect(ctx, providerID); err != nil {
		w.logger.Warn("providerconnect: submit post-connect workflow failed",
			"provider_id", providerID,
			"error", err,
		)
	}
}
