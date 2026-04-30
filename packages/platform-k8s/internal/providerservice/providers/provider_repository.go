package providers

import (
	"code-code.internal/platform-k8s/internal/platform/providerstate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists provider aggregate roots.
// Re-exported from the shared providerstate package so that in-package code
// continues to compile without import changes.
type Store = providerstate.Store

// ProviderRepository is the Postgres-backed implementation of Store.
type ProviderRepository = providerstate.ProviderRepository

// NewProviderRepository creates a provider repository backed by Postgres.
func NewProviderRepository(pool *pgxpool.Pool) (*ProviderRepository, error) {
	return providerstate.NewProviderRepository(pool)
}

// ProviderStateProjection wraps a provider aggregate for read-side dispatch paths.
type ProviderStateProjection = providerstate.ProviderProjection
