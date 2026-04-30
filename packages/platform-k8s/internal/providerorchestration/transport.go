package providerorchestration

import (
	"fmt"

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
	activities := &Activities{auth: server.auth, provider: server.provider, connect: server.connect}
	worker.RegisterActivityWithOptions(activities.ConnectAPIKeyProvider, activity.RegisterOptions{Name: connectAPIKeyProviderActivityName})
	worker.RegisterActivityWithOptions(activities.ConnectCLIOAuthProvider, activity.RegisterOptions{Name: connectCLIOAuthProviderActivityName})
	worker.RegisterActivityWithOptions(activities.ReauthorizeProvider, activity.RegisterOptions{Name: reauthorizeProviderActivityName})
	worker.RegisterActivityWithOptions(activities.GetProvider, activity.RegisterOptions{Name: getProviderActivityName})
	worker.RegisterActivityWithOptions(activities.DeleteCredential, activity.RegisterOptions{Name: deleteCredentialActivityName})
	worker.RegisterActivityWithOptions(activities.GetProviderConnectSession, activity.RegisterOptions{Name: getProviderConnectSessionActivityName})
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

func cloneManagementSurfaceCatalogs(items []*managementv1.ProviderSurfaceModelCatalog) []*managementv1.ProviderSurfaceModelCatalog {
	if len(items) == 0 {
		return nil
	}
	out := make([]*managementv1.ProviderSurfaceModelCatalog, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, proto.Clone(item).(*managementv1.ProviderSurfaceModelCatalog))
	}
	return out
}

func providerConnectSurfaceCatalogs(items []*managementv1.ProviderSurfaceModelCatalog) []*providerconnect.ProviderModelCatalogInput {
	if len(items) == 0 {
		return nil
	}
	out := make([]*providerconnect.ProviderModelCatalogInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &providerconnect.ProviderModelCatalogInput{
			SurfaceID: item.GetSurfaceId(),
			Models:    cloneProviderModels(item.GetModels()),
		})
	}
	return out
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
	return &managementv1.ProviderConnectSessionView{
		SessionId:        view.GetSessionId(),
		OauthSessionId:   view.GetOauthSessionId(),
		Phase:            managementSessionPhaseFromProviderConnect(view.GetPhase()),
		DisplayName:      view.GetDisplayName(),
		AuthorizationUrl: view.GetAuthorizationUrl(),
		UserCode:         view.GetUserCode(),
		Message:          view.GetMessage(),
		ErrorMessage:     view.GetErrorMessage(),
		Provider:         managementProviderViewFromProviderConnect(view.GetProvider()),
		AddMethod:        managementAddMethodFromProviderConnect(view.GetAddMethod()),
		VendorId:         view.GetVendorId(),
		CliId:            view.GetCliId(),
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
		ProductInfoId:        view.GetProductInfoId(),
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
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

func cloneProviderModels(items []*providerv1.ProviderModelCatalogEntry) []*providerv1.ProviderModelCatalogEntry {
	if len(items) == 0 {
		return nil
	}
	out := make([]*providerv1.ProviderModelCatalogEntry, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, proto.Clone(item).(*providerv1.ProviderModelCatalogEntry))
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
