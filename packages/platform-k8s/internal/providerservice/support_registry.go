package providerservice

import (
	"context"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

// SurfaceRegistry provides access to provider surface definitions.
type SurfaceRegistry interface {
	List(ctx context.Context) ([]*supportv1.Surface, error)
	Get(ctx context.Context, surfaceID string) (*supportv1.Surface, error)
}
