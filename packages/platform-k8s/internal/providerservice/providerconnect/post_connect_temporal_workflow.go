package providerconnect

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code-code.internal/platform-k8s/internal/platform/httpauth"
	"code-code.internal/platform-k8s/internal/platform/temporalruntime"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	TemporalTaskQueue = "platform-provider-service"

	PostConnectWorkflowName       = "platform.provider.postConnect"
	providerPostConnectActivity   = "platform.provider.postConnect.trigger"
	defaultProviderPostConnectURL = "http://platform-provider-service.code-code.svc.cluster.local:8080/internal/actions"
)

type TemporalPostConnectWorkflowRuntimeConfig struct {
	Client                  client.Client
	TaskQueue               string
	PlatformNamespace       string
	ProviderHTTPBaseURL     string
	ProviderHTTPActionToken string
}

type TemporalPostConnectWorkflowRuntime struct {
	client                  client.Client
	taskQueue               string
	providerHTTPBaseURL     string
	providerHTTPActionToken string
}

func NewTemporalPostConnectWorkflowRuntime(config TemporalPostConnectWorkflowRuntimeConfig) (*TemporalPostConnectWorkflowRuntime, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: temporal client is nil")
	}
	taskQueue := strings.TrimSpace(config.TaskQueue)
	if taskQueue == "" {
		taskQueue = TemporalTaskQueue
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.ProviderHTTPBaseURL), "/")
	if baseURL == "" {
		namespace := strings.TrimSpace(config.PlatformNamespace)
		if namespace == "" {
			baseURL = defaultProviderPostConnectURL
		} else {
			baseURL = fmt.Sprintf("http://platform-provider-service.%s.svc.cluster.local:8080/internal/actions", namespace)
		}
	}
	return &TemporalPostConnectWorkflowRuntime{
		client:                  config.Client,
		taskQueue:               taskQueue,
		providerHTTPBaseURL:     baseURL,
		providerHTTPActionToken: strings.TrimSpace(config.ProviderHTTPActionToken),
	}, nil
}

func (r *TemporalPostConnectWorkflowRuntime) SubmitPostConnect(ctx context.Context, providerID string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return fmt.Errorf("platformk8s/providerconnect: provider id is empty")
	}
	_, err := r.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       "provider-post-connect-" + temporalruntime.IDPart(providerID, "provider"),
		TaskQueue:                r.taskQueue,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, PostConnectWorkflowName, PostConnectInput{ProviderID: providerID})
	return err
}

func (r *TemporalPostConnectWorkflowRuntime) ProviderHTTPBaseURL() string {
	if r == nil {
		return ""
	}
	return r.providerHTTPBaseURL
}

func (r *TemporalPostConnectWorkflowRuntime) Register(worker worker.Worker) error {
	if worker == nil {
		return fmt.Errorf("platformk8s/providerconnect: temporal worker is nil")
	}
	worker.RegisterWorkflowWithOptions(PostConnectWorkflow, workflow.RegisterOptions{Name: PostConnectWorkflowName})
	activities := &PostConnectActivities{
		baseURL:     r.providerHTTPBaseURL,
		actionToken: r.providerHTTPActionToken,
	}
	worker.RegisterActivityWithOptions(activities.Trigger, activity.RegisterOptions{Name: providerPostConnectActivity})
	return nil
}

type PostConnectInput struct {
	ProviderID string
}

type PostConnectActivityInput struct {
	ProviderID string
	Action     string
	Trigger    string
}

func PostConnectWorkflow(ctx workflow.Context, input PostConnectInput) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: 30 * time.Second,
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	actions := []string{
		"discover-provider-catalogs",
		"bind-provider-catalogs",
		"submit-provider-observability-probe",
	}
	for _, action := range actions {
		if err := workflow.ExecuteActivity(ctx, providerPostConnectActivity, PostConnectActivityInput{
			ProviderID: input.ProviderID,
			Action:     action,
			Trigger:    "connect",
		}).Get(ctx, nil); err != nil {
			return err
		}
	}
	return nil
}

type PostConnectActivities struct {
	baseURL     string
	actionToken string
}

func (a *PostConnectActivities) Trigger(ctx context.Context, input PostConnectActivityInput) error {
	action := strings.TrimSpace(input.Action)
	if action == "" || strings.Contains(action, "/") {
		return fmt.Errorf("platformk8s/providerconnect: post-connect action is invalid")
	}
	body := fmt.Sprintf(`{"provider_ids":[%q],"trigger":%q}`, strings.TrimSpace(input.ProviderID), strings.TrimSpace(input.Trigger))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.baseURL, "/")+"/"+action, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	httpauth.SetBearerAuthorization(request, a.actionToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("platformk8s/providerconnect: post-connect %q returned %s", action, response.Status)
	}
	return nil
}
