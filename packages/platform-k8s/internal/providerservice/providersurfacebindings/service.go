package providersurfacebindings

import "github.com/jackc/pgx/v5/pgxpool"

// Service owns provider surface binding CRUD and management-facing projections.
type Service struct {
	repository *Repository
}

// NewService creates one provider surface binding service.
func NewService(pool *pgxpool.Pool) (*Service, error) {
	repository, err := NewRepository(pool)
	if err != nil {
		return nil, err
	}
	return &Service{repository: repository}, nil
}
