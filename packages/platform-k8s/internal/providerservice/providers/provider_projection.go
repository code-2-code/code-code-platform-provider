package providers

import (
	"context"

	"code-code.internal/platform-k8s/internal/platform/providerstate"
)

// ListProviderProjections returns all provider projections.
func ListProviderProjections(ctx context.Context, repository Store) ([]ProviderStateProjection, error) {
	return providerstate.ListProviderProjections(ctx, repository)
}

// FindProviderProjectionBySurfaceID finds a provider projection by surface ID.
func FindProviderProjectionBySurfaceID(ctx context.Context, repository Store, surfaceID string) (*ProviderStateProjection, error) {
	return providerstate.FindProviderProjectionBySurfaceID(ctx, repository, surfaceID)
}
