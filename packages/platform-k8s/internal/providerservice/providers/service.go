package providers

import (
	"context"
	"fmt"

	providerv1 "code-code.internal/go-contract/provider/v1"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repository Store
	surfaces   SurfaceRegistry
}

type SurfaceRegistry interface {
	Get(context.Context, string) (*providerv1.ProviderSurface, error)
}

type Config struct {
	StatePool *pgxpool.Pool
	Surfaces  SurfaceRegistry
}

func NewService(config Config) (*Service, error) {
	switch {
	case config.StatePool == nil:
		return nil, fmt.Errorf("platformk8s/providers: state pool is nil")
	}
	repository, err := NewProviderRepository(config.StatePool)
	if err != nil {
		return nil, err
	}
	return &Service{
		repository: repository,
		surfaces:   config.Surfaces,
	}, nil
}
