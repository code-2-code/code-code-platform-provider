package providers

import (
	"context"
	"slices"
	"strings"

	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
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
	return s.decorate(ctx, providerProjectionFromProvider(provider)), nil
}

func (s *Service) listProviderProjections(ctx context.Context) ([]*ProviderProjection, error) {
	providers, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*ProviderProjection, 0, len(providers))
	for _, provider := range providers {
		items = append(items, providerProjectionFromProvider(provider))
	}
	slices.SortFunc(items, compareProviderProjections)
	return s.decorateAll(ctx, items), nil
}

func providerViews(projections []*ProviderProjection) []*managementv1.ProviderView {
	items := make([]*managementv1.ProviderView, 0, len(projections))
	for _, projection := range projections {
		items = append(items, projection.Proto())
	}
	return items
}

func (s *Service) decorate(ctx context.Context, projection *ProviderProjection) *ProviderProjection {
	items := s.decorateAll(ctx, []*ProviderProjection{projection})
	if len(items) == 0 {
		return projection
	}
	return items[0]
}

func (s *Service) decorateAll(ctx context.Context, projections []*ProviderProjection) []*ProviderProjection {
	projections = newProviderIconRuntime(s.vendors, s.cliDefs).Apply(ctx, projections)
	return newCredentialSubjectSummaryRuntime(s.credentials).Apply(ctx, projections)
}
