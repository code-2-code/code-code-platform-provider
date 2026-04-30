package providers

import (
	"context"
	"slices"
	"strings"

	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func (s *Service) List(ctx context.Context) ([]*managementv1.ProviderView, error) {
	projections, err := s.listProviderProjections(ctx)
	if err != nil {
		return nil, err
	}
	return providerViews(projections), nil
}

func (s *Service) Get(ctx context.Context, providerID string) (*managementv1.ProviderView, error) {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return projection.Proto(), nil
}

func (s *Service) getProviderProjection(ctx context.Context, providerID string) (*ProviderProjection, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	provider, err := s.repository.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return s.providerProjectionFromProvider(ctx, provider), nil
}

func (s *Service) listProviderProjections(ctx context.Context) ([]*ProviderProjection, error) {
	providers, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*ProviderProjection, 0, len(providers))
	for _, provider := range providers {
		items = append(items, s.providerProjectionFromProvider(ctx, provider))
	}
	slices.SortFunc(items, compareProviderProjections)
	return items, nil
}

func (s *Service) providerProjectionFromProvider(ctx context.Context, provider *providerv1.Provider) *ProviderProjection {
	return providerProjectionFromProvider(provider, s.providerSurfaceForProjection(ctx, provider))
}

func (s *Service) providerSurfaceForProjection(ctx context.Context, provider *providerv1.Provider) *providerv1.ProviderSurface {
	if s == nil || s.surfaces == nil || provider == nil {
		return nil
	}
	surfaceID := strings.TrimSpace(provider.GetSurfaceId())
	if surfaceID == "" {
		return nil
	}
	surface, err := s.surfaces.Get(ctx, surfaceID)
	if err != nil {
		return nil
	}
	return surface
}

func providerViews(projections []*ProviderProjection) []*managementv1.ProviderView {
	items := make([]*managementv1.ProviderView, 0, len(projections))
	for _, projection := range projections {
		items = append(items, projection.Proto())
	}
	return items
}
