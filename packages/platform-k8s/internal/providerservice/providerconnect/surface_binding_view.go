package providerconnect

import (
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

// ProviderSurfaceBindingStatusView is the providerconnect-owned provider surface binding status projection.
type ProviderSurfaceBindingStatusView struct {
	Phase  ProviderSurfaceBindingPhase
	Reason string
}

func (v *ProviderSurfaceBindingStatusView) GetPhase() ProviderSurfaceBindingPhase {
	if v == nil {
		return ProviderSurfaceBindingPhaseUnspecified
	}
	return v.Phase
}

func (v *ProviderSurfaceBindingStatusView) GetReason() string {
	if v == nil {
		return ""
	}
	return v.Reason
}

// ProviderSurfaceBindingView is the providerconnect-owned provider surface binding projection.
type ProviderSurfaceBindingView struct {
	DisplayName          string
	SurfaceID            string
	ProviderCredentialID string
	Runtime              *providerv1.ProviderSurfaceRuntime
	Status               *ProviderSurfaceBindingStatusView
	VendorID             string
	ProviderID           string
	ProviderDisplayName  string
}

func (v *ProviderSurfaceBindingView) GetDisplayName() string {
	if v == nil {
		return ""
	}
	return v.DisplayName
}

func (v *ProviderSurfaceBindingView) GetSurfaceId() string {
	if v == nil {
		return ""
	}
	return v.SurfaceID
}

func (v *ProviderSurfaceBindingView) GetProviderCredentialId() string {
	if v == nil {
		return ""
	}
	return v.ProviderCredentialID
}

func (v *ProviderSurfaceBindingView) GetRuntime() *providerv1.ProviderSurfaceRuntime {
	if v == nil {
		return nil
	}
	return v.Runtime
}

func (v *ProviderSurfaceBindingView) GetStatus() *ProviderSurfaceBindingStatusView {
	if v == nil {
		return nil
	}
	return v.Status
}

func (v *ProviderSurfaceBindingView) GetVendorId() string {
	if v == nil {
		return ""
	}
	return v.VendorID
}

func (v *ProviderSurfaceBindingView) GetProviderId() string {
	if v == nil {
		return ""
	}
	return v.ProviderID
}

func (v *ProviderSurfaceBindingView) GetProviderDisplayName() string {
	if v == nil {
		return ""
	}
	return v.ProviderDisplayName
}

func cloneProviderSurfaceBindingView(view *ProviderSurfaceBindingView) *ProviderSurfaceBindingView {
	if view == nil {
		return nil
	}
	next := *view
	if view.Runtime != nil {
		next.Runtime = proto.Clone(view.Runtime).(*providerv1.ProviderSurfaceRuntime)
	}
	if view.Status != nil {
		status := *view.Status
		next.Status = &status
	}
	return &next
}
