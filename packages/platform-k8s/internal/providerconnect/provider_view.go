package providerconnect

import (
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

// ProviderStatusView is the providerconnect-owned provider status projection.
type ProviderStatusView struct {
	Phase  ProviderPhase
	Reason string
}

func (v *ProviderStatusView) GetPhase() ProviderPhase {
	if v == nil {
		return ProviderPhaseUnspecified
	}
	return v.Phase
}

func (v *ProviderStatusView) GetReason() string {
	if v == nil {
		return ""
	}
	return v.Reason
}

// ProviderView is the providerconnect-owned provider projection.
type ProviderView struct {
	ProviderID           string
	DisplayName          string
	SurfaceID            string
	ProviderCredentialID string
	Runtime              *providerv1.ProviderSurfaceRuntime
	Status               *ProviderStatusView
	ProductInfoID        string
	VendorID             string
	ProviderDisplayName  string
}

func (v *ProviderView) GetProviderId() string {
	if v == nil {
		return ""
	}
	return v.ProviderID
}

func (v *ProviderView) GetDisplayName() string {
	if v == nil {
		return ""
	}
	return v.DisplayName
}

func (v *ProviderView) GetSurfaceId() string {
	if v == nil {
		return ""
	}
	return v.SurfaceID
}

func (v *ProviderView) GetProviderCredentialId() string {
	if v == nil {
		return ""
	}
	return v.ProviderCredentialID
}

func (v *ProviderView) GetRuntime() *providerv1.ProviderSurfaceRuntime {
	if v == nil {
		return nil
	}
	return v.Runtime
}

func (v *ProviderView) GetStatus() *ProviderStatusView {
	if v == nil {
		return nil
	}
	return v.Status
}

func (v *ProviderView) GetProductInfoId() string {
	if v == nil {
		return ""
	}
	return v.ProductInfoID
}

func (v *ProviderView) GetVendorId() string {
	if v == nil {
		return ""
	}
	return v.VendorID
}

func (v *ProviderView) GetProviderDisplayName() string {
	if v == nil {
		return ""
	}
	return v.ProviderDisplayName
}

func cloneProviderView(view *ProviderView) *ProviderView {
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
