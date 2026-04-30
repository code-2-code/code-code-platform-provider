package providerservice

import (
	"context"

	providerv1 "code-code.internal/go-contract/provider/v1"
)

// SurfaceRegistry provides access to provider surface definitions.
type SurfaceRegistry interface {
	List(ctx context.Context) ([]*providerv1.ProviderSurface, error)
	Get(ctx context.Context, surfaceID string) (*providerv1.ProviderSurface, error)
}
