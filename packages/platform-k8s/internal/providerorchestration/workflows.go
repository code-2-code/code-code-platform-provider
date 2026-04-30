package providerorchestration

import (
	"fmt"
	"time"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ProviderAPIKeyConnectWorkflowName            = "platform.providerOrchestration.connectApiKey"
	ProviderCLIOAuthConnectWorkflowName          = "platform.providerOrchestration.connectCliOAuth"
	ProviderCLIOAuthReauthorizationWorkflowName  = "platform.providerOrchestration.reauthorizeCliOAuth"
	ProviderAPIKeyAuthUpdatedWorkflowName        = "platform.providerOrchestration.apiKeyAuthUpdated"
	ProviderObservabilityAuthUpdatedWorkflowName = "platform.providerOrchestration.observabilityAuthUpdated"
	ProviderConnectSessionSyncWorkflowName       = "platform.providerOrchestration.syncConnectSession"

	connectAPIKeyProviderActivityName     = "platform.providerOrchestration.connectApiKeyProvider"
	connectCLIOAuthProviderActivityName   = "platform.providerOrchestration.connectCliOAuthProvider"
	reauthorizeProviderActivityName       = "platform.providerOrchestration.reauthorizeProvider"
	getProviderActivityName               = "platform.providerOrchestration.getProvider"
	deleteCredentialActivityName          = "platform.providerOrchestration.deleteCredential"
	getProviderConnectSessionActivityName = "platform.providerOrchestration.getConnectSession"
)

type APIKeyConnectWorkflowInput struct {
	CredentialID string
	DisplayName  string
	VendorID     string
	BaseURL      string
	Protocol     apiprotocolv1.Protocol
	Catalogs     []*managementv1.ProviderSurfaceModelCatalog
	Compensate   bool
}

type CLIOAuthConnectWorkflowInput struct {
	DisplayName string
	CLIID       string
}

type CLIOAuthReauthorizationWorkflowInput struct {
	ProviderID string
}

type ProviderUpdatedWorkflowInput struct {
	ProviderID string
}

type ProviderConnectSessionWorkflowInput struct {
	SessionID string
}

func ProviderAPIKeyConnectWorkflow(ctx workflow.Context, input APIKeyConnectWorkflowInput) (*managementv1.ConnectProviderResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, providerActivityOptions())
	var response managementv1.ConnectProviderResponse
	err := workflow.ExecuteActivity(ctx, connectAPIKeyProviderActivityName, input).Get(ctx, &response)
	if err == nil {
		return &response, nil
	}
	if input.Compensate && input.CredentialID != "" {
		disconnected, _ := workflow.NewDisconnectedContext(ctx)
		var deleteResult string
		if deleteErr := workflow.ExecuteActivity(disconnected, deleteCredentialActivityName, input.CredentialID).Get(disconnected, &deleteResult); deleteErr != nil {
			return nil, fmt.Errorf("connect provider failed: %w; credential compensation failed: %v", err, deleteErr)
		}
	}
	return nil, err
}

func ProviderCLIOAuthConnectWorkflow(ctx workflow.Context, input CLIOAuthConnectWorkflowInput) (*managementv1.ConnectProviderResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, providerActivityOptions())
	var response managementv1.ConnectProviderResponse
	if err := workflow.ExecuteActivity(ctx, connectCLIOAuthProviderActivityName, input).Get(ctx, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func ProviderCLIOAuthReauthorizationWorkflow(ctx workflow.Context, input CLIOAuthReauthorizationWorkflowInput) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, providerActivityOptions())
	var response managementv1.UpdateProviderAuthenticationResponse
	if err := workflow.ExecuteActivity(ctx, reauthorizeProviderActivityName, input).Get(ctx, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func ProviderAPIKeyAuthUpdatedWorkflow(ctx workflow.Context, input ProviderUpdatedWorkflowInput) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, providerActivityOptions())
	var provider managementv1.ProviderView
	if err := workflow.ExecuteActivity(ctx, getProviderActivityName, input.ProviderID).Get(ctx, &provider); err != nil {
		return nil, err
	}
	return &managementv1.UpdateProviderAuthenticationResponse{
		Outcome: &managementv1.UpdateProviderAuthenticationResponse_Provider{Provider: &provider},
	}, nil
}

func ProviderObservabilityAuthUpdatedWorkflow(ctx workflow.Context, input ProviderUpdatedWorkflowInput) (*managementv1.UpdateProviderObservabilityAuthenticationResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, providerActivityOptions())
	var provider managementv1.ProviderView
	if err := workflow.ExecuteActivity(ctx, getProviderActivityName, input.ProviderID).Get(ctx, &provider); err != nil {
		return nil, err
	}
	return &managementv1.UpdateProviderObservabilityAuthenticationResponse{Provider: &provider}, nil
}

func ProviderConnectSessionSyncWorkflow(ctx workflow.Context, input ProviderConnectSessionWorkflowInput) (*managementv1.GetProviderConnectSessionResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, providerActivityOptions())
	var response managementv1.GetProviderConnectSessionResponse
	if err := workflow.ExecuteActivity(ctx, getProviderConnectSessionActivityName, input.SessionID).Get(ctx, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func providerActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: 20 * time.Second,
			MaximumAttempts: 3,
		},
	}
}
