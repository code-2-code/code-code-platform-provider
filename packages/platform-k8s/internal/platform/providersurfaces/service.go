package providersurfaces

import (
	"context"
	"fmt"
	"slices"
	"strings"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces/registry"
	vendorsupport "code-code.internal/platform-k8s/internal/platform/vendors/support"
)

// Service exposes the effective provider surface read path.
type Service struct {
	surfaces map[string]*supportv1.Surface
}

// NewService creates one provider surface service.
func NewService() (*Service, error) {
	vendors, err := vendorsupport.NewManagementService()
	if err != nil {
		return nil, err
	}
	items, err := vendors.List(context.Background())
	if err != nil {
		return nil, err
	}
	surfaces := make(map[string]*supportv1.Surface)
	for _, vendor := range items {
		for _, surface := range vendor.GetSurfaces() {
			surfaceID := strings.TrimSpace(surface.GetSurfaceId())
			if surfaceID == "" {
				return nil, fmt.Errorf("platformk8s/providersurfaces: surface id is empty")
			}
			if _, exists := surfaces[surfaceID]; exists {
				return nil, fmt.Errorf("platformk8s/providersurfaces: provider surface %q is already registered", surfaceID)
			}
			surfaces[surfaceID] = cloneSurface(surface)
		}
	}
	surfaces[registry.SurfaceIDCustomAPIKey] = customAPIKeySurface()
	return &Service{
		surfaces: surfaces,
	}, nil
}

func customAPIKeySurface() *supportv1.Surface {
	return &supportv1.Surface{
		SurfaceId:     registry.SurfaceIDCustomAPIKey,
		ProductInfoId: "custom-api-key",
		Spec: &supportv1.Surface_Api{
			Api: &supportv1.ApiSurface{},
		},
		EgressPolicyId: "custom.api",
		AuthPolicyId:   "protocol.default.api-key",
	}
}

// Get returns one effective provider surface by stable identity.
func (s *Service) Get(ctx context.Context, surfaceID string) (*supportv1.Surface, error) {
	surface, ok := s.surfaces[strings.TrimSpace(surfaceID)]
	if !ok {
		return nil, fmt.Errorf("platformk8s/providersurfaces: provider surface %q not found", surfaceID)
	}
	return cloneSurface(surface), nil
}

// List returns all effective provider surfaces.
func (s *Service) List(ctx context.Context) ([]*supportv1.Surface, error) {
	items := make([]*supportv1.Surface, 0, len(s.surfaces))
	for _, item := range s.surfaces {
		items = append(items, cloneSurface(item))
	}
	slices.SortFunc(items, func(left, right *supportv1.Surface) int {
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
