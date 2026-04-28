package providerservice

import (
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"google.golang.org/protobuf/proto"
)

func providerConnectSurfaceBindingToTransport(view *providerconnect.ProviderSurfaceBindingView) *providerservicev1.ProviderSurfaceBindingView {
	if view == nil {
		return nil
	}
	out := &providerservicev1.ProviderSurfaceBindingView{
		DisplayName:          view.DisplayName,
		SurfaceId:            view.SurfaceID,
		ProviderCredentialId: view.ProviderCredentialID,
		ProviderId:           view.ProviderID,
		Status:               providerConnectSurfaceBindingStatusToTransport(view.Status),
		VendorId:             view.VendorID,
		ProviderDisplayName:  view.ProviderDisplayName,
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
		if out.Status == nil {
			out.Status = surfaceBindingStatusFromRuntimeToService(runtime)
		}
	}
	return out
}

func providerConnectOAuthSummaryFieldFromTransport(view *managementv1.CredentialSubjectSummaryFieldView) *providerconnect.CredentialSubjectSummaryFieldView {
	if view == nil {
		return nil
	}
	return &providerconnect.CredentialSubjectSummaryFieldView{
		FieldID: strings.TrimSpace(view.GetFieldId()),
		Label:   strings.TrimSpace(view.GetLabel()),
		Value:   strings.TrimSpace(view.GetValue()),
	}
}

func providerConnectOAuthSummaryFieldToTransport(view *providerconnect.CredentialSubjectSummaryFieldView) *providerservicev1.CredentialSubjectSummaryFieldView {
	if view == nil {
		return nil
	}
	return &providerservicev1.CredentialSubjectSummaryFieldView{
		FieldId: view.FieldID,
		Label:   view.Label,
		Value:   view.Value,
	}
}

func providerConnectSurfaceBindingStatusFromTransport(view *managementv1.ProviderSurfaceBindingStatus) *providerconnect.ProviderSurfaceBindingStatusView {
	if view == nil {
		return nil
	}
	return &providerconnect.ProviderSurfaceBindingStatusView{
		Phase:  providerConnectSurfaceBindingPhaseFromTransport(view.GetPhase()),
		Reason: strings.TrimSpace(view.GetReason()),
	}
}

func providerConnectSurfaceBindingStatusToTransport(view *providerconnect.ProviderSurfaceBindingStatusView) *providerservicev1.ProviderSurfaceBindingStatus {
	if view == nil {
		return nil
	}
	return &providerservicev1.ProviderSurfaceBindingStatus{
		Phase:  providerConnectSurfaceBindingPhaseToTransport(view.GetPhase()),
		Reason: view.GetReason(),
	}
}

func providerConnectSurfaceBindingPhaseFromTransport(phase managementv1.ProviderSurfaceBindingPhase) providerconnect.ProviderSurfaceBindingPhase {
	switch phase {
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY:
		return providerconnect.ProviderSurfaceBindingPhaseReady
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG:
		return providerconnect.ProviderSurfaceBindingPhaseInvalidConfig
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_REFRESHING:
		return providerconnect.ProviderSurfaceBindingPhaseRefreshing
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_STALE:
		return providerconnect.ProviderSurfaceBindingPhaseStale
	case managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_ERROR:
		return providerconnect.ProviderSurfaceBindingPhaseError
	default:
		return providerconnect.ProviderSurfaceBindingPhaseUnspecified
	}
}

func providerConnectSurfaceBindingPhaseToTransport(phase providerconnect.ProviderSurfaceBindingPhase) providerservicev1.ProviderSurfaceBindingPhase {
	switch phase {
	case providerconnect.ProviderSurfaceBindingPhaseReady:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY
	case providerconnect.ProviderSurfaceBindingPhaseInvalidConfig:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG
	case providerconnect.ProviderSurfaceBindingPhaseRefreshing:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_REFRESHING
	case providerconnect.ProviderSurfaceBindingPhaseStale:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_STALE
	case providerconnect.ProviderSurfaceBindingPhaseError:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_ERROR
	default:
		return providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_UNSPECIFIED
	}
}

func surfaceBindingStatusFromRuntimeToService(runtime *providerv1.ProviderSurfaceRuntime) *providerservicev1.ProviderSurfaceBindingStatus {
	if runtime == nil {
		return &providerservicev1.ProviderSurfaceBindingStatus{
			Phase:  providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG,
			Reason: "provider surface runtime is nil",
		}
	}
	if err := providerv1.ValidateProviderSurfaceRuntime(runtime); err != nil {
		return &providerservicev1.ProviderSurfaceBindingStatus{
			Phase:  providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_INVALID_CONFIG,
			Reason: err.Error(),
		}
	}
	return &providerservicev1.ProviderSurfaceBindingStatus{
		Phase: providerservicev1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY,
	}
}

func cloneProviderModelCatalogModels(items []*providerv1.ProviderModelCatalogEntry) []*providerv1.ProviderModelCatalogEntry {
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
