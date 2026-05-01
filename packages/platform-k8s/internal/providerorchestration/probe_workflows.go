package providerorchestration

import (
	"fmt"
	"strings"
	"time"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ProviderQuotaProbeSweepWorkflowName        = "platform.providerOrchestration.sweepQuotaProbes"
	ProviderModelCatalogProbeSweepWorkflowName = "platform.providerOrchestration.sweepModelCatalogProbes"

	listProbeProviderIDsActivityName     = "platform.providerOrchestration.listProbeProviderIDs"
	runQuotaProbeTaskActivityName        = "platform.providerOrchestration.runQuotaProbeTask"
	runModelCatalogProbeTaskActivityName = "platform.providerOrchestration.runModelCatalogProbeTask"
)

type providerProbeKind string

const (
	providerProbeKindModelCatalog providerProbeKind = "model-catalog"
	providerProbeKindQuota        providerProbeKind = "quota"
)

type ProviderProbeTaskInput struct {
	ProviderID string
	Trigger    providerservicev1.ProviderObservabilityProbeTrigger
}

type ProviderProbeSweepWorkflowInput struct{}

type ProviderProbePreflightInput struct {
	ProviderID string
	Kind       providerProbeKind
	Trigger    providerservicev1.ProviderObservabilityProbeTrigger
}

type ProviderProbePreflightDecision struct {
	ProviderID      string
	Kind            providerProbeKind
	ShouldRun       bool
	ProbeID         string
	Message         string
	Outcome         string
	MinimumInterval time.Duration
}

type ProviderProbeStatusInput struct {
	ProviderID      string
	Kind            providerProbeKind
	ProbeID         string
	Outcome         string
	Message         string
	MinimumInterval time.Duration
	Trigger         providerservicev1.ProviderObservabilityProbeTrigger
}

func ProviderQuotaProbeSweepWorkflow(ctx workflow.Context, _ ProviderProbeSweepWorkflowInput) error {
	return sweepProviderProbes(ctx, providerProbeKindQuota)
}

func ProviderModelCatalogProbeSweepWorkflow(ctx workflow.Context, _ ProviderProbeSweepWorkflowInput) error {
	return sweepProviderProbes(ctx, providerProbeKindModelCatalog)
}

func sweepProviderProbes(ctx workflow.Context, kind providerProbeKind) error {
	ctx = workflow.WithActivityOptions(ctx, probeActivityOptions())
	var providerIDs []string
	if err := workflow.ExecuteActivity(ctx, listProbeProviderIDsActivityName).Get(ctx, &providerIDs); err != nil {
		return err
	}
	activityName := runQuotaProbeTaskActivityName
	if kind == providerProbeKindModelCatalog {
		activityName = runModelCatalogProbeTaskActivityName
	}
	futures := make([]workflow.Future, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		providerID = strings.TrimSpace(providerID)
		if providerID == "" {
			continue
		}
		futures = append(futures, workflow.ExecuteActivity(ctx, activityName, ProviderProbeTaskInput{
			ProviderID: providerID,
			Trigger:    providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_SCHEDULE,
		}))
	}
	for _, future := range futures {
		_ = future.Get(ctx, nil)
	}
	return nil
}

func providerProbeID(providerID string, kind providerProbeKind) string {
	return strings.TrimSpace(providerID) + ":" + string(kind)
}

func providerProbeSweepWorkflowID(kind providerProbeKind) string {
	return "provider:" + string(kind) + ":sweep"
}

func probeActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: 5 * time.Second,
			MaximumAttempts: 2,
		},
	}
}

func quotaProbeOutcome(outcome string) providerservicev1.ProviderOAuthObservabilityProbeOutcome {
	switch strings.TrimSpace(outcome) {
	case "throttled":
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_THROTTLED
	case "auth_blocked":
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_AUTH_BLOCKED
	case "unsupported":
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSUPPORTED
	case "failed":
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_FAILED
	case "executed":
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED
	default:
		return providerservicev1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSPECIFIED
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func validateProbeKind(kind providerProbeKind) error {
	switch kind {
	case providerProbeKindModelCatalog, providerProbeKindQuota:
		return nil
	default:
		return fmt.Errorf("platformk8s/providerorchestration: unsupported probe kind %q", kind)
	}
}

func quotaProbeResponseFromDecision(decision *ProviderProbePreflightDecision) *managementv1.ProbeProviderObservabilityResponse {
	if decision == nil {
		return &managementv1.ProbeProviderObservabilityResponse{}
	}
	return &managementv1.ProbeProviderObservabilityResponse{
		ProviderId: decision.ProviderID,
		Message:    decision.Message,
		ProbeId:    decision.ProbeID,
		Outcome:    quotaProbeOutcome(decision.Outcome),
	}
}

func modelCatalogProbeResponseFromDecision(decision *ProviderProbePreflightDecision) *managementv1.ProbeProviderModelCatalogResponse {
	if decision == nil {
		return &managementv1.ProbeProviderModelCatalogResponse{}
	}
	return &managementv1.ProbeProviderModelCatalogResponse{
		ProviderId: decision.ProviderID,
		Message:    decision.Message,
		ProbeId:    decision.ProbeID,
	}
}
