package providerstate

import (
	"context"
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

// ProviderProjection wraps a provider aggregate for read-side dispatch paths.
type ProviderProjection struct {
	Provider *providerv1.Provider
}

// ListProviderProjections returns all provider projections from the store.
func ListProviderProjections(ctx context.Context, repository Store) ([]ProviderProjection, error) {
	providers, err := repository.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ProviderProjection, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		items = append(items, ProviderProjection{Provider: provider})
	}
	return items, nil
}

// FindProviderProjectionBySurfaceID finds a provider projection by surface ID.
func FindProviderProjectionBySurfaceID(ctx context.Context, repository Store, surfaceID string) (*ProviderProjection, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerstate: provider surface id is empty")
	}
	items, err := ListProviderProjections(ctx, repository)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.Provider.GetSurfaceId()) == surfaceID {
			found := item
			return &found, nil
		}
	}
	return nil, domainerror.NewNotFound("platformk8s/providerstate: provider surface %q not found", surfaceID)
}
