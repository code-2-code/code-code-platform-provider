package providerobservability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func NewService(config Config) (*Service, error) {
	capabilities, err := normalizeCapabilities(config.Capabilities)
	if err != nil {
		return nil, err
	}
	switch {
	case config.ProviderSurfaceBindings == nil:
		return nil, fmt.Errorf("platformk8s/providerobservability: provider surface binding lister is nil")
	}
	return &Service{
		providerSurfaceBindings: config.ProviderSurfaceBindings,
		capabilities:            capabilities,
	}, nil
}

func normalizeCapabilities(items []Capability) ([]Capability, error) {
	out := make([]Capability, 0, len(items))
	seen := map[OwnerKind]struct{}{}
	for _, item := range items {
		if item == nil {
			continue
		}
		kind := OwnerKind(strings.TrimSpace(string(item.OwnerKind())))
		if kind == "" {
			return nil, fmt.Errorf("platformk8s/providerobservability: capability owner kind is empty")
		}
		if _, ok := seen[kind]; ok {
			return nil, fmt.Errorf("platformk8s/providerobservability: duplicate capability owner kind %q", kind)
		}
		seen[kind] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("platformk8s/providerobservability: no observability capabilities configured")
	}
	return out, nil
}

func (s *Service) ProbeProvider(
	ctx context.Context,
	providerID string,
	trigger Trigger,
) (*managementv1.ProbeProviderObservabilityResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("platformk8s/providerobservability: service is nil")
	}
	target, err := s.findProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if target.capability != nil {
		probeTarget := target.probeTarget()
		result, err := target.capability.ProbeProvider(ctx, probeTarget, trigger)
		if err != nil {
			return nil, err
		}
		return buildResponse(probeTarget, result), nil
	}
	return &managementv1.ProbeProviderObservabilityResponse{
		ProviderId: target.providerID,
		Outcome:    managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSUPPORTED,
		Message:    "provider has no supported observability owner",
	}, nil
}

type providerProbeTarget struct {
	providerID               string
	providerSurfaceBindingID string
	ownerID                  string
	capability               Capability
}

func (t providerProbeTarget) probeTarget() ProbeTarget {
	ownerKind := OwnerKind("")
	if t.capability != nil {
		ownerKind = t.capability.OwnerKind()
	}
	return ProbeTarget{
		ProviderID:               t.providerID,
		ProviderSurfaceBindingID: t.providerSurfaceBindingID,
		OwnerKind:                ownerKind,
		OwnerID:                  t.ownerID,
	}
}

func (s *Service) findProvider(ctx context.Context, providerID string) (*providerProbeTarget, error) {
	trimmedID := strings.TrimSpace(providerID)
	if trimmedID == "" {
		return nil, fmt.Errorf("platformk8s/providerobservability: provider id is empty")
	}
	items, err := s.providerSurfaceBindings.ListProviderSurfaceBindings(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, item := range items {
		if item == nil || strings.TrimSpace(item.GetProviderId()) != trimmedID {
			continue
		}
		found = true
		for _, capability := range s.capabilities {
			ownerID, ok := capability.Supports(item)
			ownerID = strings.TrimSpace(ownerID)
			if ok && ownerID != "" {
				return &providerProbeTarget{
					providerID:               trimmedID,
					providerSurfaceBindingID: strings.TrimSpace(item.GetSurfaceId()),
					ownerID:                  ownerID,
					capability:               capability,
				}, nil
			}
		}
	}
	if found {
		return &providerProbeTarget{providerID: trimmedID}, nil
	}
	return nil, domainerror.NewNotFound("platformk8s/providerobservability: provider %q not found", trimmedID)
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
