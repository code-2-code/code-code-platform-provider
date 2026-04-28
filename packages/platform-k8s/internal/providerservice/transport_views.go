package providerservice

import (
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func providerViewsToService(items []*managementv1.ProviderView) []*providerservicev1.ProviderView {
	out := make([]*providerservicev1.ProviderView, 0, len(items))
	for _, item := range items {
		if next := providerViewToService(item); next != nil {
			out = append(out, next)
		}
	}
	return out
}

func providerViewToService(view *managementv1.ProviderView) *providerservicev1.ProviderView {
	if view == nil {
		return nil
	}
	out := &providerservicev1.ProviderView{
		ProviderId:           view.GetProviderId(),
		DisplayName:          view.GetDisplayName(),
		VendorId:             view.GetVendorId(),
		ProviderCredentialId: view.GetProviderCredentialId(),
		IconUrl:              view.GetIconUrl(),
		ModelCatalog:         cloneProviderModelCatalog(view.GetModelCatalog()),
	}
	for _, surface := range view.GetSurfaces() {
		if next := providerSurfaceBindingViewToService(surface); next != nil {
			out.Surfaces = append(out.Surfaces, next)
		}
	}
	for _, item := range view.GetCredentialSubjectSummary() {
		if next := oauthSummaryFieldToService(item); next != nil {
			out.CredentialSubjectSummary = append(out.CredentialSubjectSummary, next)
		}
	}
	return out
}

func cloneProviderModelCatalog(catalog *providerv1.ProviderModelCatalog) *providerv1.ProviderModelCatalog {
	if catalog == nil {
		return nil
	}
	return proto.Clone(catalog).(*providerv1.ProviderModelCatalog)
}

func providerSurfaceBindingViewsToService(items []*managementv1.ProviderSurfaceBindingView) []*providerservicev1.ProviderSurfaceBindingView {
	out := make([]*providerservicev1.ProviderSurfaceBindingView, 0, len(items))
	for _, item := range items {
		if next := providerSurfaceBindingViewToService(item); next != nil {
			out = append(out, next)
		}
	}
	return out
}

func providerSurfaceBindingViewToService(view *managementv1.ProviderSurfaceBindingView) *providerservicev1.ProviderSurfaceBindingView {
	if view == nil {
		return nil
	}
	out := &providerservicev1.ProviderSurfaceBindingView{
		DisplayName:          view.GetDisplayName(),
		SurfaceId:            view.GetSurfaceId(),
		ProviderCredentialId: view.GetProviderCredentialId(),
		Status:               surfaceBindingStatusToService(view.GetStatus()),
		VendorId:             view.GetVendorId(),
		ProviderId:           view.GetProviderId(),
		ProviderDisplayName:  view.GetProviderDisplayName(),
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
	}
	return out
}

func oauthSummaryFieldToService(view *managementv1.CredentialSubjectSummaryFieldView) *providerservicev1.CredentialSubjectSummaryFieldView {
	if view == nil {
		return nil
	}
	return &providerservicev1.CredentialSubjectSummaryFieldView{
		FieldId: view.GetFieldId(),
		Label:   view.GetLabel(),
		Value:   view.GetValue(),
	}
}

func surfaceBindingStatusToService(view *managementv1.ProviderSurfaceBindingStatus) *providerservicev1.ProviderSurfaceBindingStatus {
	if view == nil {
		return nil
	}
	return &providerservicev1.ProviderSurfaceBindingStatus{
		Phase:  surfaceBindingPhaseToService(view.GetPhase()),
		Reason: view.GetReason(),
	}
}

func surfaceBindingPhaseToService(phase managementv1.ProviderSurfaceBindingPhase) providerservicev1.ProviderSurfaceBindingPhase {
	switch phase {
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_REFRESHING:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_REFRESHING
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_STALE:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_STALE
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_ERROR:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_ERROR
	default:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_UNSPECIFIED
	}
}

func updateProviderAuthenticationResponseToService(response *managementv1.UpdateProviderAuthenticationResponse) *providerservicev1.UpdateProviderAuthenticationResponse {
	if response == nil {
		return &providerservicev1.UpdateProviderAuthenticationResponse{}
	}
	switch outcome := response.GetOutcome().(type) {
	case *managementv1.UpdateProviderAuthenticationResponse_Provider:
		return &providerservicev1.UpdateProviderAuthenticationResponse{Outcome: &providerservicev1.UpdateProviderAuthenticationResponse_Provider{Provider: providerViewToService(outcome.Provider)}}
	case *managementv1.UpdateProviderAuthenticationResponse_Session:
		return &providerservicev1.UpdateProviderAuthenticationResponse{Outcome: &providerservicev1.UpdateProviderAuthenticationResponse_Session{Session: providerConnectSessionToService(outcome.Session)}}
	default:
		return &providerservicev1.UpdateProviderAuthenticationResponse{}
	}
}

func providerConnectSessionToService(view *managementv1.ProviderConnectSessionView) *providerservicev1.ProviderConnectSessionView {
	if view == nil {
		return nil
	}
	return &providerservicev1.ProviderConnectSessionView{
		SessionId:        view.GetSessionId(),
		OauthSessionId:   view.GetOauthSessionId(),
		Phase:            connectSessionPhaseToService(view.GetPhase()),
		DisplayName:      view.GetDisplayName(),
		AuthorizationUrl: view.GetAuthorizationUrl(),
		UserCode:         view.GetUserCode(),
		Message:          view.GetMessage(),
		ErrorMessage:     view.GetErrorMessage(),
		Provider:         providerViewToService(view.GetProvider()),
		AddMethod:        addMethodToService(view.GetAddMethod()),
		VendorId:         view.GetVendorId(),
		CliId:            view.GetCliId(),
	}
}

func connectSessionPhaseToService(phase managementv1.ProviderConnectSessionPhase) providerservicev1.ProviderConnectSessionPhase {
	switch phase {
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_PENDING:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_PENDING
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_AWAITING_USER:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_AWAITING_USER
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_PROCESSING:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_PROCESSING
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_SUCCEEDED:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_SUCCEEDED
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_FAILED:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_FAILED
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_EXPIRED:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_EXPIRED
	case managementv1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_CANCELED:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_CANCELED
	default:
		return providerservicev1.ProviderConnectSessionPhase_PROVIDER_CONNECT_SESSION_PHASE_UNSPECIFIED
	}
}

func addMethodToService(method managementv1.ProviderAddMethod) providerservicev1.ProviderAddMethod {
	switch method {
	case managementv1.ProviderAddMethod_PROVIDER_ADD_METHOD_API_KEY:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_API_KEY
	case managementv1.ProviderAddMethod_PROVIDER_ADD_METHOD_CLI_OAUTH:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_CLI_OAUTH
	default:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_UNSPECIFIED
	}
}
