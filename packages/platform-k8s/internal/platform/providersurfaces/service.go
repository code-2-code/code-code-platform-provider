package providersurfaces

import (
	"context"
	"fmt"
	"slices"

	providerv1 "code-code.internal/go-contract/provider/v1"
	surfaceregistry "code-code.internal/platform-k8s/internal/platform/providersurfaces/registry"
)

// Service exposes the effective provider surface read path.
type Service struct {
	builtins   map[string]*providerv1.ProviderSurface
}

// NewService creates one provider surface service.
func NewService() (*Service, error) {
	builtins := make(map[string]*providerv1.ProviderSurface)
	for _, item := range surfaceregistry.List() {
		builtins[item.GetSurfaceId()] = cloneSurface(item)
	}
	return &Service{
		builtins:   builtins,
	}, nil
}

// Get returns one effective provider surface by stable identity.
func (s *Service) Get(ctx context.Context, surfaceID string) (*providerv1.ProviderSurface, error) {
	surface, ok := s.builtins[surfaceID]
	if !ok {
		return nil, fmt.Errorf("platformk8s/providersurfaces: provider surface %q not found", surfaceID)
	}
	return cloneSurface(surface), nil
}

// List returns all effective provider surfaces.
func (s *Service) List(ctx context.Context) ([]*providerv1.ProviderSurface, error) {
	items := make([]*providerv1.ProviderSurface, 0, len(s.builtins))
	for _, item := range s.builtins {
		items = append(items, cloneSurface(item))
	}
	slices.SortFunc(items, func(left, right *providerv1.ProviderSurface) int {
		switch {
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
