package providerorchestration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	orchestrationv1 "code-code.internal/go-contract/platform/orchestration/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/resourcemeta"
	"code-code.internal/platform-k8s/internal/platform/temporalruntime"
	"code-code.internal/platform-k8s/internal/providerconnect"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const TemporalTaskQueue = "platform-provider-orchestration-service"

type Config struct {
	TemporalClient client.Client
	TaskQueue      string
	Auth           authv1.AuthServiceClient
	Provider       providerservicev1.ProviderServiceClient
	Connect        *providerconnect.Service
	Logger         *slog.Logger
}

type Server struct {
	orchestrationv1.UnimplementedProviderOrchestrationServiceServer

	temporal  client.Client
	taskQueue string
	auth      authv1.AuthServiceClient
	provider  providerservicev1.ProviderServiceClient
	connect   *providerconnect.Service
	logger    *slog.Logger
}

func NewServer(config Config) (*Server, error) {
	if config.TemporalClient == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: temporal client is nil")
	}
	if config.Auth == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: auth client is nil")
	}
	if config.Provider == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider client is nil")
	}
	if config.Connect == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: provider connect service is nil")
	}
	taskQueue := strings.TrimSpace(config.TaskQueue)
	if taskQueue == "" {
		taskQueue = TemporalTaskQueue
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		temporal:  config.TemporalClient,
		taskQueue: taskQueue,
		auth:      config.Auth,
		provider:  config.Provider,
		connect:   config.Connect,
		logger:    logger,
	}, nil
}

func (s *Server) ConnectProvider(ctx context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "connect provider request is required")
	}
	switch request.GetAuthMaterial().(type) {
	case *managementv1.ConnectProviderRequest_ApiKey:
		return s.connectAPIKeyProvider(ctx, request)
	case *managementv1.ConnectProviderRequest_CliOauth:
		return s.connectCLIOAuthProvider(ctx, request)
	default:
		return nil, status.Error(codes.InvalidArgument, "provider auth material is required")
	}
}

func (s *Server) connectAPIKeyProvider(ctx context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	material := request.GetApiKey()
	if strings.TrimSpace(material.GetApiKey()) == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key is required")
	}
	displayName := providerDisplayName(request)
	credentialID, err := resourcemeta.EnsureResourceID("", displayName, providerCredentialFallback(request))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate credential id: %v", err)
	}
	created, err := s.auth.CreateAPIKeyCredential(ctx, &authv1.CreateAPIKeyCredentialRequest{
		CredentialId: credentialID,
		DisplayName:  displayName,
		Purpose:      credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_DATA_PLANE.String(),
		ApiKey:       strings.TrimSpace(material.GetApiKey()),
	})
	if err != nil {
		return nil, err
	}
	credentialID = strings.TrimSpace(created.GetCredential().GetCredentialId())
	if credentialID == "" {
		return nil, status.Error(codes.Internal, "created credential id is empty")
	}
	input := APIKeyConnectWorkflowInput{
		CredentialID: credentialID,
		DisplayName:  request.GetDisplayName(),
		SurfaceID:    request.GetSurfaceId(),
		BaseURL:      material.GetBaseUrl(),
		Protocol:     material.GetProtocol(),
		Models:       cloneProviderModels(material.GetModels()),
		Compensate:   true,
	}
	var response managementv1.ConnectProviderResponse
	err = s.executeWorkflow(ctx, "provider-connect-api-key-"+temporalruntime.IDPart(credentialID, "credential"), ProviderAPIKeyConnectWorkflowName, input, &response)
	if err != nil {
		return nil, err
	}
	s.submitProviderPostConnect(ctx, response.GetProvider().GetProviderId())
	return &response, nil
}

func (s *Server) connectCLIOAuthProvider(ctx context.Context, request *managementv1.ConnectProviderRequest) (*managementv1.ConnectProviderResponse, error) {
	input := CLIOAuthConnectWorkflowInput{
		DisplayName: request.GetDisplayName(),
		SurfaceID:   request.GetSurfaceId(),
	}
	var response managementv1.ConnectProviderResponse
	err := s.executeWorkflow(ctx, "provider-connect-cli-"+temporalruntime.IDPart(request.GetSurfaceId()+"-"+request.GetDisplayName()+"-"+time.Now().UTC().Format("20060102150405.000000000"), "surface"), ProviderCLIOAuthConnectWorkflowName, input, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Server) GetProviderConnectSession(ctx context.Context, request *managementv1.GetProviderConnectSessionRequest) (*managementv1.GetProviderConnectSessionResponse, error) {
	if request == nil || strings.TrimSpace(request.GetSessionId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	input := ProviderConnectSessionWorkflowInput{SessionID: request.GetSessionId()}
	var response managementv1.GetProviderConnectSessionResponse
	if err := s.executeWorkflow(ctx, "provider-connect-session-sync-"+temporalruntime.IDPart(request.GetSessionId()+"-"+time.Now().UTC().Format("20060102150405.000000000"), "session"), ProviderConnectSessionSyncWorkflowName, input, &response); err != nil {
		return nil, err
	}
	if session := response.GetSession(); session.GetPhase() == providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_SUCCEEDED {
		s.submitProviderPostConnect(ctx, session.GetProvider().GetProviderId())
	}
	return &response, nil
}

func (s *Server) UpdateProviderAuthentication(ctx context.Context, request *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	if request == nil || strings.TrimSpace(request.GetProviderId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_id is required")
	}
	switch request.GetAuthMaterial().(type) {
	case *managementv1.UpdateProviderAuthenticationRequest_ApiKey:
		return s.updateAPIKeyAuthentication(ctx, request)
	case *managementv1.UpdateProviderAuthenticationRequest_CliOauth:
		input := CLIOAuthReauthorizationWorkflowInput{ProviderID: request.GetProviderId()}
		var response managementv1.UpdateProviderAuthenticationResponse
		err := s.executeWorkflow(ctx, "provider-reauth-cli-"+temporalruntime.IDPart(request.GetProviderId()+"-"+time.Now().UTC().Format("20060102150405.000000000"), "provider"), ProviderCLIOAuthReauthorizationWorkflowName, input, &response)
		if err != nil {
			return nil, err
		}
		return &response, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "provider auth material is required")
	}
}

func (s *Server) updateAPIKeyAuthentication(ctx context.Context, request *managementv1.UpdateProviderAuthenticationRequest) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	provider, err := s.getProvider(ctx, request.GetProviderId())
	if err != nil {
		return nil, err
	}
	credentialID := strings.TrimSpace(provider.GetProviderCredentialId())
	if credentialID == "" {
		return nil, status.Error(codes.FailedPrecondition, "provider does not reference a credential")
	}
	material := request.GetApiKey()
	if strings.TrimSpace(material.GetApiKey()) == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key is required")
	}
	if _, err := s.auth.UpdateAPIKeyCredential(ctx, &authv1.UpdateAPIKeyCredentialRequest{
		CredentialId: credentialID,
		DisplayName:  strings.TrimSpace(provider.GetDisplayName()),
		Purpose:      credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_DATA_PLANE.String(),
		ApiKey:       strings.TrimSpace(material.GetApiKey()),
	}); err != nil {
		return nil, err
	}
	input := ProviderUpdatedWorkflowInput{ProviderID: provider.GetProviderId()}
	var response managementv1.UpdateProviderAuthenticationResponse
	if err := s.executeWorkflow(ctx, "provider-auth-updated-"+temporalruntime.IDPart(provider.GetProviderId()+"-"+time.Now().UTC().Format("20060102150405.000000000"), "provider"), ProviderAPIKeyAuthUpdatedWorkflowName, input, &response); err != nil {
		return nil, err
	}
	s.submitProviderPostConnect(ctx, response.GetProvider().GetProviderId())
	return &response, nil
}

func (s *Server) UpdateProviderObservabilityAuthentication(ctx context.Context, request *managementv1.UpdateProviderObservabilityAuthenticationRequest) (*managementv1.UpdateProviderObservabilityAuthenticationResponse, error) {
	if request == nil || strings.TrimSpace(request.GetProviderId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_id is required")
	}
	provider, err := s.getProvider(ctx, request.GetProviderId())
	if err != nil {
		return nil, err
	}
	material := request.GetSessionMaterial()
	if material == nil || len(material.GetValues()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "session_material values are required")
	}
	credentialID := observabilityCredentialID(provider.GetProviderId())
	if _, err := s.auth.UpdateSessionCredential(ctx, &authv1.UpdateSessionCredentialRequest{
		CredentialId: credentialID,
		DisplayName:  strings.TrimSpace(provider.GetDisplayName()) + " Observability",
		Purpose:      credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_MANAGEMENT_PLANE.String(),
		SchemaId:     material.GetSchemaId(),
		RequiredKeys: append([]string(nil), material.GetRequiredKeys()...),
		Values:       cloneStringMap(material.GetValues()),
		MergeValues:  true,
	}); err != nil {
		return nil, err
	}
	input := ProviderUpdatedWorkflowInput{ProviderID: provider.GetProviderId()}
	var response managementv1.UpdateProviderObservabilityAuthenticationResponse
	if err := s.executeWorkflow(ctx, "provider-observability-auth-updated-"+temporalruntime.IDPart(provider.GetProviderId()+"-"+time.Now().UTC().Format("20060102150405.000000000"), "provider"), ProviderObservabilityAuthUpdatedWorkflowName, input, &response); err != nil {
		return nil, err
	}
	s.submitProviderPostConnect(ctx, response.GetProvider().GetProviderId())
	return &response, nil
}

func (s *Server) ProbeProviderObservability(ctx context.Context, request *managementv1.ProbeProviderObservabilityRequest) (*managementv1.ProbeProviderObservabilityResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "probe provider quota request is required")
	}
	providerIDs, err := s.probeProviderIDs(ctx, request.GetProviderId(), request.GetProviderIds())
	if err != nil {
		return nil, err
	}
	if len(providerIDs) == 0 {
		workflowID := providerProbeSweepWorkflowID(providerProbeKindQuota)
		if err := s.startWorkflow(ctx, workflowID, ProviderQuotaProbeSweepWorkflowName, ProviderProbeSweepWorkflowInput{}); err != nil {
			return nil, err
		}
		return &managementv1.ProbeProviderObservabilityResponse{Message: "provider quota sweep started"}, nil
	}
	activities := s.probeActivities()
	if len(providerIDs) == 1 {
		response, err := activities.RunQuotaProbeTask(ctx, ProviderProbeTaskInput{ProviderID: providerIDs[0], Trigger: request.GetTrigger()})
		if err != nil {
			return nil, err
		}
		return response, nil
	}
	for _, providerID := range providerIDs {
		if _, err := activities.RunQuotaProbeTask(ctx, ProviderProbeTaskInput{ProviderID: providerID, Trigger: request.GetTrigger()}); err != nil {
			return nil, err
		}
	}
	return &managementv1.ProbeProviderObservabilityResponse{
		ProviderIds: providerIDs,
		Message:     "provider quota probe tasks completed",
	}, nil
}

func (s *Server) ProbeProviderModelCatalog(ctx context.Context, request *managementv1.ProbeProviderModelCatalogRequest) (*managementv1.ProbeProviderModelCatalogResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "probe provider model catalog request is required")
	}
	providerIDs, err := s.probeProviderIDs(ctx, request.GetProviderId(), request.GetProviderIds())
	if err != nil {
		return nil, err
	}
	if len(providerIDs) == 0 {
		workflowID := providerProbeSweepWorkflowID(providerProbeKindModelCatalog)
		if err := s.startWorkflow(ctx, workflowID, ProviderModelCatalogProbeSweepWorkflowName, ProviderProbeSweepWorkflowInput{}); err != nil {
			return nil, err
		}
		return &managementv1.ProbeProviderModelCatalogResponse{Message: "provider model catalog sweep started"}, nil
	}
	activities := s.probeActivities()
	if len(providerIDs) == 1 {
		response, err := activities.RunModelCatalogProbeTask(ctx, ProviderProbeTaskInput{ProviderID: providerIDs[0]})
		if err != nil {
			return nil, err
		}
		return response, nil
	}
	for _, providerID := range providerIDs {
		if _, err := activities.RunModelCatalogProbeTask(ctx, ProviderProbeTaskInput{ProviderID: providerID}); err != nil {
			return nil, err
		}
	}
	return &managementv1.ProbeProviderModelCatalogResponse{
		ProviderIds: providerIDs,
		Message:     "provider model catalog probe tasks completed",
	}, nil
}

func (s *Server) getProvider(ctx context.Context, providerID string) (*managementv1.ProviderView, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_id is required")
	}
	response, err := s.provider.ListProviders(ctx, &providerservicev1.ListProvidersRequest{})
	if err != nil {
		return nil, err
	}
	for _, item := range response.GetItems() {
		if strings.TrimSpace(item.GetProviderId()) != providerID {
			continue
		}
		out := &managementv1.ProviderView{}
		if err := transcodeProto(item, out); err != nil {
			return nil, status.Errorf(codes.Internal, "transcode provider: %v", err)
		}
		return out, nil
	}
	return nil, status.Errorf(codes.NotFound, "provider %q not found", providerID)
}

func (s *Server) executeWorkflow(ctx context.Context, workflowID string, workflowName string, input any, out any) error {
	run, err := s.temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    strings.TrimSpace(workflowID),
		TaskQueue:             s.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}, workflowName, input)
	if err != nil {
		return err
	}
	return run.Get(ctx, out)
}

func (s *Server) startWorkflow(ctx context.Context, workflowID string, workflowName string, input any) error {
	_, err := s.temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       strings.TrimSpace(workflowID),
		TaskQueue:                s.taskQueue,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, workflowName, input)
	return err
}

func (s *Server) submitProviderPostConnect(ctx context.Context, providerID string) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return
	}
	if err := s.startWorkflow(ctx, providerPostConnectWorkflowID(providerID), ProviderPostConnectWorkflowName, ProviderUpdatedWorkflowInput{ProviderID: providerID}); err != nil {
		s.logger.Warn("provider post-connect workflow start failed",
			"provider_id", providerID,
			"error", err,
		)
	}
}

func providerPostConnectWorkflowID(providerID string) string {
	return "provider-post-connect-" + temporalruntime.IDPart(providerID, "provider")
}

func (s *Server) probeProviderIDs(ctx context.Context, providerID string, providerIDs []string) ([]string, error) {
	ids := normalizedProviderIDs(providerIDs)
	if len(ids) == 0 && strings.TrimSpace(providerID) != "" {
		ids = []string{strings.TrimSpace(providerID)}
	}
	return ids, nil
}

func (s *Server) probeActivities() *Activities {
	return &Activities{auth: s.auth, provider: s.provider, connect: s.connect}
}

func normalizedProviderIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func providerDisplayName(request *managementv1.ConnectProviderRequest) string {
	if displayName := strings.TrimSpace(request.GetDisplayName()); displayName != "" {
		return displayName
	}
	if surfaceID := strings.TrimSpace(request.GetSurfaceId()); surfaceID != "" {
		return surfaceID
	}
	return "Provider"
}

func providerCredentialFallback(request *managementv1.ConnectProviderRequest) string {
	if surfaceID := strings.TrimSpace(request.GetSurfaceId()); surfaceID != "" {
		return surfaceID
	}
	return "custom-provider"
}

func observabilityCredentialID(providerID string) string {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ""
	}
	return providerID + "-observability"
}
