package providersurfacebindings

import (
	"context"
	"fmt"

	providerv1 "code-code.internal/go-contract/provider/v1"
	provideraggregates "code-code.internal/platform-k8s/internal/providerservice/providers"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	providers provideraggregates.Store
}

func NewRepository(pool *pgxpool.Pool) (*Repository, error) {
	providers, err := provideraggregates.NewProviderRepository(pool)
	if err != nil {
		return nil, err
	}
	return &Repository{providers: providers}, nil
}

func (r *Repository) ListProviders(ctx context.Context) ([]*providerv1.Provider, error) {
	if r == nil || r.providers == nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: repository is not initialized")
	}
	return r.providers.List(ctx)
}

func (r *Repository) UpdateProvider(ctx context.Context, providerID string, mutate func(*providerv1.Provider) error) (*providerv1.Provider, error) {
	if r == nil || r.providers == nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: repository is not initialized")
	}
	return r.providers.Update(ctx, providerID, mutate)
}

func (r *Repository) GetProvider(ctx context.Context, providerID string) (*providerv1.Provider, error) {
	if r == nil || r.providers == nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: repository is not initialized")
	}
	return r.providers.Get(ctx, providerID)
}

func (r *Repository) UpsertProvider(ctx context.Context, provider *providerv1.Provider) (*providerv1.Provider, error) {
	if r == nil || r.providers == nil {
		return nil, fmt.Errorf("platformk8s/providersurfacebindings: repository is not initialized")
	}
	return r.providers.Upsert(ctx, provider)
}
