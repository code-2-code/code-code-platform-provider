package providerorchestration

import (
	"fmt"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerconnect"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func RegisterTemporalWorkflows(worker worker.Worker, server *Server) error {
	if worker == nil {
		return fmt.Errorf("platformk8s/providerorchestration: temporal worker is nil")
	}
	if server == nil {
		return fmt.Errorf("platformk8s/providerorchestration: server is nil")
	}
	worker.RegisterWorkflowWithOptions(ProviderAPIKeyConnectWorkflow, workflow.RegisterOptions{Name: ProviderAPIKeyConnectWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderCLIOAuthConnectWorkflow, workflow.RegisterOptions{Name: ProviderCLIOAuthConnectWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderCLIOAuthReauthorizationWorkflow, workflow.RegisterOptions{Name: ProviderCLIOAuthReauthorizationWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderAPIKeyAuthUpdatedWorkflow, workflow.RegisterOptions{Name: ProviderAPIKeyAuthUpdatedWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderObservabilityAuthUpdatedWorkflow, workflow.RegisterOptions{Name: ProviderObservabilityAuthUpdatedWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderConnectSessionSyncWorkflow, workflow.RegisterOptions{Name: ProviderConnectSessionSyncWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderPostConnectWorkflow, workflow.RegisterOptions{Name: ProviderPostConnectWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderQuotaProbeSweepWorkflow, workflow.RegisterOptions{Name: ProviderQuotaProbeSweepWorkflowName})
	worker.RegisterWorkflowWithOptions(ProviderModelCatalogProbeSweepWorkflow, workflow.RegisterOptions{Name: ProviderModelCatalogProbeSweepWorkflowName})
	activities := &Activities{auth: server.auth, provider: server.provider, connect: server.connect}
	worker.RegisterActivityWithOptions(activities.ConnectAPIKeyProvider, activity.RegisterOptions{Name: connectAPIKeyProviderActivityName})
	worker.RegisterActivityWithOptions(activities.ConnectCLIOAuthProvider, activity.RegisterOptions{Name: connectCLIOAuthProviderActivityName})
	worker.RegisterActivityWithOptions(activities.ReauthorizeProvider, activity.RegisterOptions{Name: reauthorizeProviderActivityName})
	worker.RegisterActivityWithOptions(activities.GetProvider, activity.RegisterOptions{Name: getProviderActivityName})
	worker.RegisterActivityWithOptions(activities.DeleteCredential, activity.RegisterOptions{Name: deleteCredentialActivityName})
	worker.RegisterActivityWithOptions(activities.GetProviderConnectSession, activity.RegisterOptions{Name: getProviderConnectSessionActivityName})
	worker.RegisterActivityWithOptions(activities.ListProbeProviderIDs, activity.RegisterOptions{Name: listProbeProviderIDsActivityName})
	worker.RegisterActivityWithOptions(activities.RunQuotaProbeTask, activity.RegisterOptions{Name: runQuotaProbeTaskActivityName})
	worker.RegisterActivityWithOptions(activities.RunModelCatalogProbeTask, activity.RegisterOptions{Name: runModelCatalogProbeTaskActivityName})
	return nil
}

func transcodeProto(src proto.Message, dst proto.Message) error {
	if src == nil || dst == nil {
		return nil
	}
	body, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshal proto: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, dst); err != nil {
		return fmt.Errorf("unmarshal proto: %w", err)
	}
	return nil
}

func providerConnectSurfaceModels(surfaceID string, models []*providerv1.ProviderModel) []*providerconnect.SurfaceModelInput {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" || len(models) == 0 {
		return nil
	}
	return []*providerconnect.SurfaceModelInput{{
		SurfaceID: surfaceID,
		Models:    cloneProviderModels(models),
	}}
}

func managementConnectResponseFromResult(result *providerconnect.ConnectResult) *managementv1.ConnectProviderResponse {
	if result == nil {
		return &managementv1.ConnectProviderResponse{}
	}
	if result.GetSession() != nil {
		return &managementv1.ConnectProviderResponse{
			Outcome: &managementv1.ConnectProviderResponse_Session{
				Session: managementSessionViewFromProviderConnect(result.GetSession()),
			},
		}
	}
	return &managementv1.ConnectProviderResponse{
		Outcome: &managementv1.ConnectProviderResponse_Provider{
			Provider: managementProviderViewFromProviderConnect(result.GetProvider()),
		},
	}
}

func managementSessionViewFromProviderConnect(view *providerconnect.SessionView) *managementv1.ProviderConnectSessionView {
	if view == nil {
		return nil
	}
	provider := managementProviderViewFromProviderConnect(view.GetProvider())
	surfaceID := ""
	if provider != nil {
		surfaceID = provider.GetSurfaceId()
	}
	return &managementv1.ProviderConnectSessionView{
		SessionId:        view.GetSessionId(),
		OauthSessionId:   view.GetOauthSessionId(),
		Phase:            managementSessionPhaseFromProviderConnect(view.GetPhase()),
		DisplayName:      view.GetDisplayName(),
		AuthorizationUrl: view.GetAuthorizationUrl(),
		UserCode:         view.GetUserCode(),
		Message:          view.GetMessage(),
		ErrorMessage:     view.GetErrorMessage(),
		Provider:         provider,
		AddMethod:        managementAddMethodFromProviderConnect(view.GetAddMethod()),
		SurfaceId:        surfaceID,
	}
}

func managementProviderViewFromProviderConnect(view *providerconnect.ProviderView) *managementv1.ProviderView {
	if view == nil {
		return nil
	}
	out := &managementv1.ProviderView{
		ProviderId:           view.GetProviderId(),
		DisplayName:          view.GetDisplayName(),
		SurfaceId:            view.GetSurfaceId(),
		ProviderCredentialId: view.GetProviderCredentialId(),
		Models:               cloneProviderModels(view.GetModels()),
		Endpoints:            cloneProviderEndpoints(view.GetEndpoints()),
		Status: &managementv1.ProviderStatus{
			Phase:  managementProviderPhaseFromProviderConnect(view.GetStatus().GetPhase()),
			Reason: view.GetStatus().GetReason(),
		},
	}
	return out
}

func managementAddMethodFromProviderConnect(value providerconnect.AddMethod) providerservicev1.ProviderAddMethod {
	switch value {
	case providerconnect.AddMethodAPIKey:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_API_KEY
	case providerconnect.AddMethodCLIOAuth:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_CLI_OAUTH
	default:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_UNSPECIFIED
	}
}

func managementSessionPhaseFromProviderConnect(value providerconnect.SessionPhase) providerservicev1.ProviderConnectSessionPhase {
	switch value {
	case providerconnect.SessionPhasePending:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_PENDING
	case providerconnect.SessionPhaseAwaitingUser:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_AWAITING_USER
	case providerconnect.SessionPhaseProcessing:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_PROCESSING
	case providerconnect.SessionPhaseSucceeded:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_SUCCEEDED
	case providerconnect.SessionPhaseFailed:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_FAILED
	case providerconnect.SessionPhaseExpired:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_EXPIRED
	case providerconnect.SessionPhaseCanceled:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_CANCELED
	default:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_UNSPECIFIED
	}
}

func managementProviderPhaseFromProviderConnect(value providerconnect.ProviderPhase) providerservicev1.ProviderPhase {
	switch value {
	case providerconnect.ProviderPhaseReady:
		return providerservicev1.ProviderPhase_PROVIDER_PHASE_READY
	case providerconnect.ProviderPhaseInvalidConfig:
		return providerservicev1.ProviderPhase_PROVIDER_PHASE_INVALID_CONFIG
	case providerconnect.ProviderPhaseRefreshing:
		return providerservicev1.ProviderPhase_PROVIDER_PHASE_REFRESHING
	case providerconnect.ProviderPhaseStale:
		return providerservicev1.ProviderPhase_PROVIDER_PHASE_STALE
	case providerconnect.ProviderPhaseError:
		return providerservicev1.ProviderPhase_PROVIDER_PHASE_ERROR
	default:
		return providerservicev1.ProviderPhase_PROVIDER_PHASE_UNSPECIFIED
	}
}

func cloneProviderModels(items []*providerv1.ProviderModel) []*providerv1.ProviderModel {
	if len(items) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderModel, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, proto.Clone(item).(*providerv1.ProviderModel))
	}
	return out
}

func cloneProviderEndpoints(items []*providerv1.ProviderEndpoint) []*providerv1.ProviderEndpoint {
	if len(items) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderEndpoint, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, proto.Clone(item).(*providerv1.ProviderEndpoint))
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
