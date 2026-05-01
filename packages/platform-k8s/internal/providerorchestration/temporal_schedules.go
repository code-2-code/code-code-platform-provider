package providerorchestration

import (
	"context"
	"time"

	"code-code.internal/platform-k8s/internal/platform/temporalruntime"
	"go.temporal.io/sdk/client"
)

func EnsureTemporalSchedules(ctx context.Context, temporalClient client.Client, taskQueue string) error {
	if err := temporalruntime.EnsureIntervalSchedule(ctx, temporalClient, temporalruntime.IntervalSchedule{
		ID:         "provider-quota-probe-sweep",
		WorkflowID: providerProbeSweepWorkflowID(providerProbeKindQuota),
		Workflow:   ProviderQuotaProbeSweepWorkflowName,
		TaskQueue:  taskQueue,
		Every:      time.Minute,
	}); err != nil {
		return err
	}
	return temporalruntime.EnsureIntervalSchedule(ctx, temporalClient, temporalruntime.IntervalSchedule{
		ID:         "provider-model-catalog-probe-sweep",
		WorkflowID: providerProbeSweepWorkflowID(providerProbeKindModelCatalog),
		Workflow:   ProviderModelCatalogProbeSweepWorkflowName,
		TaskQueue:  taskQueue,
		Every:      time.Minute,
		Offset:     30 * time.Second,
	})
}
