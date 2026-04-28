package providerservice

import (
	"context"
	"fmt"
	"time"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/temporalruntime"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	TemporalTaskQueue = "platform-provider-service"

	ProviderObservabilityScheduleWorkflowName = "platform.provider.observabilitySchedule"
	probeProviderObservabilityActivityName    = "platform.provider.probeObservability"
)

func RegisterTemporalWorkflows(worker worker.Worker, server *Server) error {
	if worker == nil {
		return fmt.Errorf("platformk8s/providerservice: temporal worker is nil")
	}
	if server == nil {
		return fmt.Errorf("platformk8s/providerservice: server is nil")
	}
	worker.RegisterWorkflowWithOptions(ProviderObservabilityScheduleWorkflow, workflow.RegisterOptions{Name: ProviderObservabilityScheduleWorkflowName})
	activities := &TemporalActivities{server: server}
	worker.RegisterActivityWithOptions(activities.ProbeProviderObservability, activity.RegisterOptions{Name: probeProviderObservabilityActivityName})
	return nil
}

func EnsureTemporalSchedules(ctx context.Context, client client.Client, taskQueue string) error {
	return temporalruntime.EnsureIntervalSchedule(ctx, client, temporalruntime.IntervalSchedule{
		ID:         "provider-observability-schedule",
		WorkflowID: "provider-observability-schedule",
		Workflow:   ProviderObservabilityScheduleWorkflowName,
		TaskQueue:  taskQueue,
		Every:      3 * time.Minute,
	})
}

func ProviderObservabilityScheduleWorkflow(ctx workflow.Context) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 4 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: 30 * time.Second,
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	return workflow.ExecuteActivity(ctx, probeProviderObservabilityActivityName).Get(ctx, nil)
}

type TemporalActivities struct {
	server *Server
}

func (a *TemporalActivities) ProbeProviderObservability(ctx context.Context) error {
	if a == nil || a.server == nil {
		return fmt.Errorf("platformk8s/providerservice: temporal activity server is nil")
	}
	_, err := a.server.ProbeProviderObservability(ctx, &providerservicev1.ProbeProviderObservabilityRequest{
		Trigger: providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_SCHEDULE,
	})
	return err
}
