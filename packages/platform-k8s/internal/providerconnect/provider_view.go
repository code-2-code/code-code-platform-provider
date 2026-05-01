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
	Endpoints            []*providerv1.ProviderEndpoint
	Models               []*providerv1.ProviderModel
	Status               *ProviderStatusView
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

func (v *ProviderView) GetEndpoints() []*providerv1.ProviderEndpoint {
	if v == nil {
		return nil
	}
	return v.Endpoints
}

func (v *ProviderView) GetModels() []*providerv1.ProviderModel {
	if v == nil {
		return nil
	}
	return v.Models
}

func (v *ProviderView) GetStatus() *ProviderStatusView {
	if v == nil {
		return nil
	}
	return v.Status
}

func cloneProviderView(view *ProviderView) *ProviderView {
	if view == nil {
		return nil
	}
	next := *view
	if len(view.Endpoints) > 0 {
		next.Endpoints = make([]*providerv1.ProviderEndpoint, 0, len(view.Endpoints))
		for _, endpoint := range view.Endpoints {
			if endpoint != nil {
				next.Endpoints = append(next.Endpoints, proto.Clone(endpoint).(*providerv1.ProviderEndpoint))
			}
		}
	}
	if len(view.Models) > 0 {
		next.Models = cloneProviderModels(view.Models)
	}
	if view.Status != nil {
		status := *view.Status
		next.Status = &status
	}
	return &next
}
