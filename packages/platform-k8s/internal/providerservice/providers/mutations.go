package providers

import (
	"context"

	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func (s *Service) CreateProvider(ctx context.Context, provider *providerv1.Provider) (*managementv1.ProviderView, error) {
	if provider == nil {
		return nil, domainerror.NewValidation("platformk8s/providers: provider is nil")
	}
	next, err := s.repository.Upsert(ctx, provider)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, next.GetProviderId())
}

func (s *Service) Update(ctx context.Context, providerID string, command UpdateProviderCommand) (*managementv1.ProviderView, error) {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := s.mutationRuntime().Rename(ctx, projection, command); err != nil {
		return nil, err
	}
	return s.Get(ctx, projection.ID())
}

func (s *Service) ApplyModelCatalog(ctx context.Context, providerID string, models []*providerv1.ProviderModel) (*managementv1.ProviderView, error) {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := s.mutationRuntime().ApplyModelCatalog(ctx, projection, models); err != nil {
		return nil, err
	}
	return s.Get(ctx, projection.ID())
}

func (s *Service) ApplyProbeStatus(ctx context.Context, providerID string, kind providerv1.ProviderProbeKind, state *providerv1.ProviderProbeRunState) (*managementv1.ProviderView, error) {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := s.mutationRuntime().ApplyProbeStatus(ctx, projection, kind, state); err != nil {
		return nil, err
	}
	return s.Get(ctx, projection.ID())
}

func (s *Service) Delete(ctx context.Context, providerID string) error {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return err
	}
	return s.mutationRuntime().Delete(ctx, projection)
}
