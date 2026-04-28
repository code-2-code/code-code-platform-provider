package providerservice

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"google.golang.org/protobuf/proto"
)

func providerConnectCommandFromRequest(request *providerservicev1.ConnectProviderRequest) (*providerconnect.ConnectCommand, error) {
	if request == nil {
		return nil, domainerror.NewValidation("platformk8s/providerservice: connect provider request is nil")
	}
	addMethod, err := providerConnectAddMethodFromTransport(request.GetAddMethod())
	if err != nil {
		return nil, err
	}
	var apiKey *providerconnect.APIKeyConnectInput
	if material := request.GetApiKey(); material != nil {
		apiKey = &providerconnect.APIKeyConnectInput{
			APIKey:   material.GetApiKey(),
			BaseURL:  material.GetBaseUrl(),
			Protocol: material.GetProtocol(),
		}
		for _, item := range material.GetSurfaceModelCatalogs() {
			if item == nil {
				continue
			}
			apiKey.SurfaceModelCatalogs = append(apiKey.SurfaceModelCatalogs, &providerconnect.ProviderSurfaceBindingModelCatalogInput{
				SurfaceID: strings.TrimSpace(item.GetSurfaceId()),
				Models:    cloneProviderModelCatalogModels(item.GetModels()),
			})
		}
	}
	return providerconnect.NewConnectCommand(providerconnect.ConnectCommandInput{
		AddMethod:   addMethod,
		DisplayName: request.GetDisplayName(),
		VendorID:    request.GetVendorId(),
		CLIID:       request.GetCliId(),
		APIKey:      apiKey,
	})
}

func providerConnectAddMethodFromTransport(value providerservicev1.ProviderAddMethod) (providerconnect.AddMethod, error) {
	switch value {
	case providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_UNSPECIFIED:
		return providerconnect.AddMethodUnspecified, nil
	case providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_API_KEY:
		return providerconnect.AddMethodAPIKey, nil
	case providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_CLI_OAUTH:
		return providerconnect.AddMethodCLIOAuth, nil
	default:
		return providerconnect.AddMethodUnspecified, domainerror.NewValidation(
			"platformk8s/providerservice: unsupported add_method %s",
			value.String(),
		)
	}
}

func providerConnectAddMethodToTransport(value providerconnect.AddMethod) providerservicev1.ProviderAddMethod {
	switch value {
	case providerconnect.AddMethodAPIKey:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_API_KEY
	case providerconnect.AddMethodCLIOAuth:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_CLI_OAUTH
	default:
		return providerservicev1.ProviderAddMethod_PROVIDER_ADD_METHOD_UNSPECIFIED
	}
}

func connectProviderResponseFromResult(result *providerconnect.ConnectResult) *providerservicev1.ConnectProviderResponse {
	if result == nil {
		return &providerservicev1.ConnectProviderResponse{}
	}
	if result.Session != nil {
		return &providerservicev1.ConnectProviderResponse{Outcome: &providerservicev1.ConnectProviderResponse_Session{Session: providerConnectSessionViewToTransport(result.Session)}}
	}
	return &providerservicev1.ConnectProviderResponse{Outcome: &providerservicev1.ConnectProviderResponse_Provider{Provider: providerConnectProviderToTransport(result.Provider)}}
}

func providerConnectSessionViewToTransport(view *providerconnect.SessionView) *providerservicev1.ProviderConnectSessionView {
	if view == nil {
		return nil
	}
	return &providerservicev1.ProviderConnectSessionView{
		SessionId:        view.SessionID,
		OauthSessionId:   view.OAuthSessionID,
		Phase:            providerConnectSessionPhaseToTransport(view.Phase),
		DisplayName:      view.DisplayName,
		AuthorizationUrl: view.AuthorizationURL,
		UserCode:         view.UserCode,
		Message:          view.Message,
		ErrorMessage:     view.ErrorMessage,
		Provider:         providerConnectProviderToTransport(view.Provider),
		AddMethod:        providerConnectAddMethodToTransport(view.AddMethod),
		VendorId:         view.VendorID,
		CliId:            view.CLIID,
	}
}

func providerConnectSessionPhaseToTransport(phase providerconnect.SessionPhase) providerservicev1.ProviderConnectSessionPhase {
	switch phase {
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

func providerConnectProviderFromTransport(view *managementv1.ProviderView) *providerconnect.ProviderView {
	if view == nil {
		return nil
	}
	out := &providerconnect.ProviderView{
		ProviderID:           strings.TrimSpace(view.GetProviderId()),
		DisplayName:          strings.TrimSpace(view.GetDisplayName()),
		VendorID:             strings.TrimSpace(view.GetVendorId()),
		ProviderCredentialID: strings.TrimSpace(view.GetProviderCredentialId()),
		IconURL:              strings.TrimSpace(view.GetIconUrl()),
	}
	for _, item := range view.GetSurfaces() {
		if next := providerConnectSurfaceBindingFromTransport(item); next != nil {
			out.Surfaces = append(out.Surfaces, next)
		}
	}
	for _, item := range view.GetCredentialSubjectSummary() {
		if next := providerConnectOAuthSummaryFieldFromTransport(item); next != nil {
			out.CredentialSubjectSummary = append(out.CredentialSubjectSummary, next)
		}
	}
	return out
}

func providerConnectProviderToTransport(view *providerconnect.ProviderView) *providerservicev1.ProviderView {
	if view == nil {
		return nil
	}
	out := &providerservicev1.ProviderView{
		ProviderId:           view.ProviderID,
		DisplayName:          view.DisplayName,
		VendorId:             view.VendorID,
		ProviderCredentialId: view.ProviderCredentialID,
		IconUrl:              view.IconURL,
	}
	for _, item := range view.GetSurfaces() {
		if next := providerConnectSurfaceBindingToTransport(item); next != nil {
			out.Surfaces = append(out.Surfaces, next)
		}
	}
	for _, item := range view.GetCredentialSubjectSummary() {
		if next := providerConnectOAuthSummaryFieldToTransport(item); next != nil {
			out.CredentialSubjectSummary = append(out.CredentialSubjectSummary, next)
		}
	}
	return out
}

func providerConnectSurfaceBindingFromTransport(view *managementv1.ProviderSurfaceBindingView) *providerconnect.ProviderSurfaceBindingView {
	if view == nil {
		return nil
	}
	out := &providerconnect.ProviderSurfaceBindingView{
		DisplayName:          strings.TrimSpace(view.GetDisplayName()),
		SurfaceID:            strings.TrimSpace(view.GetSurfaceId()),
		ProviderCredentialID: strings.TrimSpace(view.GetProviderCredentialId()),
		ProviderID:           strings.TrimSpace(view.GetProviderId()),
		Status:               providerConnectSurfaceBindingStatusFromTransport(view.GetStatus()),
		VendorID:             strings.TrimSpace(view.GetVendorId()),
		ProviderDisplayName:  strings.TrimSpace(view.GetProviderDisplayName()),
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
	}
	return out
}
