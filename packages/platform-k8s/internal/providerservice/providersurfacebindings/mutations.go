package providersurfacebindings

import (
	"context"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func (s *Service) CreateProvider(ctx context.Context, provider *providerv1.Provider) (*managementv1.ProviderView, error) {
	materialized, err := s.repository.UpsertProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	return providerView(materialized), nil
}

// CreateProviderSurfaceBinding adds one surface under an existing provider aggregate.
func (s *Service) CreateProviderSurfaceBinding(ctx context.Context, providerID string, input *providerv1.ProviderSurfaceBinding) (*managementv1.ProviderSurfaceBindingView, error) {
	surface, err := NewProviderSurfaceBindingForCreate(providerID, input)
	if err != nil {
		return nil, err
	}
	provider, err := s.repository.UpdateProvider(ctx, surface.ProviderID(), func(provider *providerv1.Provider) error {
		for _, existing := range provider.GetSurfaces() {
			if strings.TrimSpace(existing.GetSurfaceId()) == surface.SurfaceID() {
				return domainerror.NewAlreadyExists("platformk8s/providersurfacebindings: provider surface binding %q already exists", surface.SurfaceID())
			}
		}
		provider.Surfaces = append(provider.Surfaces, surface.Proto())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: create surface %q: %w", surface.SurfaceID(), err)
	}
	return surfaceBindingViewWithProvider(surface, provider), nil
}

// UpdateProviderSurfaceBinding updates one surface inside its provider aggregate.
func (s *Service) UpdateProviderSurfaceBinding(ctx context.Context, surfaceID string, input *providerv1.ProviderSurfaceBinding) (*managementv1.ProviderSurfaceBindingView, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: surface id is empty")
	}
	providerID, current, err := s.findSurface(ctx, surfaceID)
	if err != nil {
		return nil, err
	}
	updated, err := NewProviderSurfaceBindingForUpdate(current, input)
	if err != nil {
		return nil, err
	}
	provider, err := s.repository.UpdateProvider(ctx, providerID, func(provider *providerv1.Provider) error {
		for index, surface := range provider.GetSurfaces() {
			if strings.TrimSpace(surface.GetSurfaceId()) == surfaceID {
				provider.Surfaces[index] = updated.Proto()
				return nil
			}
		}
		return domainerror.NewNotFound("platformk8s/providersurfacebindings: provider surface binding %q not found", surfaceID)
	})
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: update surface %q: %w", surfaceID, err)
	}
	return surfaceBindingViewWithProvider(updated, provider), nil
}

// DeleteProviderSurfaceBinding removes one surface from its provider aggregate.
func (s *Service) DeleteProviderSurfaceBinding(ctx context.Context, surfaceID string) error {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return domainerror.NewValidation("platformk8s/providersurfacebindings: surface id is empty")
	}
	providerID, _, err := s.findSurface(ctx, surfaceID)
	if err != nil {
		return err
	}
	_, err = s.repository.UpdateProvider(ctx, providerID, func(provider *providerv1.Provider) error {
		items := provider.GetSurfaces()
		for index, surface := range items {
			if strings.TrimSpace(surface.GetSurfaceId()) == surfaceID {
				provider.Surfaces = append(items[:index], items[index+1:]...)
				return nil
			}
		}
		return domainerror.NewNotFound("platformk8s/providersurfacebindings: provider surface binding %q not found", surfaceID)
	})
	return err
}

func (s *Service) findSurface(ctx context.Context, surfaceID string) (string, *SurfaceBinding, error) {
	resources, err := s.repository.ListProviders(ctx)
	if err != nil {
		return "", nil, err
	}
	for _, provider := range resources {
		for _, surface := range provider.GetSurfaces() {
			if strings.TrimSpace(surface.GetSurfaceId()) == surfaceID {
				return provider.GetProviderId(), &SurfaceBinding{value: surface, providerID: provider.GetProviderId()}, nil
			}
		}
	}
	return "", nil, domainerror.NewNotFound("platformk8s/providersurfacebindings: provider surface binding %q not found", surfaceID)
}
