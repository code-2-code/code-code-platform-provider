package providers

import (
	"context"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

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

func (s *Service) Delete(ctx context.Context, providerID string) error {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return err
	}
	return s.mutationRuntime().Delete(ctx, projection)
}

func (s *Service) UpdateAPIKeyAuthentication(
	ctx context.Context,
	providerID string,
	command UpdateAPIKeyAuthenticationCommand,
) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := s.mutationRuntime().UpdateAPIKeyAuthentication(ctx, projection, command); err != nil {
		return nil, err
	}
	next, err := s.Get(ctx, projection.ID())
	if err != nil {
		return nil, err
	}
	return &managementv1.UpdateProviderAuthenticationResponse{
		Outcome: &managementv1.UpdateProviderAuthenticationResponse_Provider{
			Provider: next,
		},
	}, nil
}

func (s *Service) UpdateObservabilityAuthentication(
	ctx context.Context,
	providerID string,
	command UpdateObservabilityAuthenticationCommand,
) (*managementv1.ProviderView, error) {
	projection, err := s.getProviderProjection(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := s.mutationRuntime().UpdateObservabilityAuthentication(ctx, projection, command); err != nil {
		return nil, err
	}
	return s.Get(ctx, projection.ID())
}
