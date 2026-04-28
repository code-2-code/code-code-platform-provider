package providers

import (
	"context"
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

type SurfaceBindingProjection struct {
	Provider *providerv1.Provider
	Surface  *providerv1.ProviderSurfaceBinding
}

func ListSurfaceBindingProjections(ctx context.Context, repository Store) ([]SurfaceBindingProjection, error) {
	providers, err := repository.List(ctx)
	if err != nil {
		return nil, err
	}
	items := []SurfaceBindingProjection{}
	for _, provider := range providers {
		for _, surface := range provider.GetSurfaces() {
			if surface == nil {
				continue
			}
			items = append(items, SurfaceBindingProjection{Provider: provider, Surface: surface})
		}
	}
	return items, nil
}

func FindSurfaceBindingProjection(ctx context.Context, repository Store, surfaceID string) (*SurfaceBindingProjection, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: provider surface binding id is empty")
	}
	items, err := ListSurfaceBindingProjections(ctx, repository)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.Surface.GetSurfaceId()) == surfaceID {
			found := item
			return &found, nil
		}
	}
	return nil, domainerror.NewNotFound("platformk8s/providers: provider surface binding %q not found", surfaceID)
}
