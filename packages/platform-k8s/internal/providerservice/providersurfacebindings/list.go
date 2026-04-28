package providersurfacebindings

import (
	"context"
	"fmt"
	"slices"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

// ListProviderSurfaceBindings returns UI-facing provider surface bindings.
func (s *Service) ListProviderSurfaceBindings(ctx context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	resources, err := s.repository.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: list providers: %w", err)
	}
	items := []*managementv1.ProviderSurfaceBindingView{}
	for _, provider := range resources {
		for _, surface := range provider.GetSurfaces() {
			if surface.GetRuntime() == nil {
				continue
			}
			items = append(items, surfaceBindingViewWithProvider(&SurfaceBinding{value: surface, providerID: provider.GetProviderId()}, provider))
		}
	}
	slices.SortFunc(items, func(left, right *managementv1.ProviderSurfaceBindingView) int {
		switch {
		case left.GetDisplayName() < right.GetDisplayName():
			return -1
		case left.GetDisplayName() > right.GetDisplayName():
			return 1
		case left.GetSurfaceId() < right.GetSurfaceId():
			return -1
		case left.GetSurfaceId() > right.GetSurfaceId():
			return 1
		default:
			return 0
		}
	})
	return items, nil
}
