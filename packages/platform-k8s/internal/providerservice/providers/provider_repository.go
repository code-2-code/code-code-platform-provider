package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	statepostgres "code-code.internal/platform-k8s/internal/platform/state/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const postgresProviderTable = "platform_providers"

// Store persists provider aggregate roots.
type Store interface {
	List(context.Context) ([]*providerv1.Provider, error)
	Get(context.Context, string) (*providerv1.Provider, error)
	Upsert(context.Context, *providerv1.Provider) (*providerv1.Provider, error)
	Update(context.Context, string, func(*providerv1.Provider) error) (*providerv1.Provider, error)
	Delete(context.Context, string) error
}

type ProviderRepository struct {
	resources *statepostgres.JSONRepository
}

func NewProviderRepository(pool *pgxpool.Pool) (*ProviderRepository, error) {
	resources, err := statepostgres.NewJSONRepository(pool, postgresProviderTable)
	if err != nil {
		return nil, err
	}
	return &ProviderRepository{resources: resources}, nil
}

func (r *ProviderRepository) List(ctx context.Context) ([]*providerv1.Provider, error) {
	if r == nil || r.resources == nil {
		return nil, fmt.Errorf("platformk8s/providers: repository is not initialized")
	}
	records, err := r.resources.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*providerv1.Provider, 0, len(records))
	for _, record := range records {
		provider, err := unmarshalProvider(record.ID, record.Payload)
		if err != nil {
			return nil, err
		}
		items = append(items, provider)
	}
	return items, nil
}

func (r *ProviderRepository) Get(ctx context.Context, providerID string) (*providerv1.Provider, error) {
	if r == nil || r.resources == nil {
		return nil, fmt.Errorf("platformk8s/providers: repository is not initialized")
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	payload, _, err := r.resources.Get(ctx, providerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, providerNotFound(providerID)
		}
		return nil, err
	}
	return unmarshalProvider(providerID, payload)
}

func (r *ProviderRepository) Upsert(ctx context.Context, provider *providerv1.Provider) (*providerv1.Provider, error) {
	if r == nil || r.resources == nil {
		return nil, fmt.Errorf("platformk8s/providers: repository is not initialized")
	}
	payload, id, normalized, err := marshalProvider(provider)
	if err != nil {
		return nil, err
	}
	if err := r.resources.Put(ctx, id, payload); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (r *ProviderRepository) Update(ctx context.Context, providerID string, mutate func(*providerv1.Provider) error) (*providerv1.Provider, error) {
	if r == nil || r.resources == nil {
		return nil, fmt.Errorf("platformk8s/providers: repository is not initialized")
	}
	if mutate == nil {
		return nil, fmt.Errorf("platformk8s/providers: mutate function is nil")
	}
	provider, err := r.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := mutate(provider); err != nil {
		return nil, err
	}
	normalized, err := normalizeProviderForRead(providerID, provider)
	if err != nil {
		return nil, err
	}
	payload, id, err := marshalNormalizedProvider(normalized)
	if err != nil {
		return nil, err
	}
	if _, err := r.resources.Update(ctx, id, payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, providerNotFound(id)
		}
		return nil, err
	}
	return normalized, nil
}

func (r *ProviderRepository) Delete(ctx context.Context, providerID string) error {
	if r == nil || r.resources == nil {
		return fmt.Errorf("platformk8s/providers: repository is not initialized")
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	if _, err := r.Get(ctx, providerID); err != nil {
		return err
	}
	return r.resources.Delete(ctx, providerID)
}

func marshalProvider(provider *providerv1.Provider) ([]byte, string, *providerv1.Provider, error) {
	normalized, err := normalizeProviderForWrite("", provider)
	if err != nil {
		return nil, "", nil, err
	}
	payload, id, err := marshalNormalizedProvider(normalized)
	if err != nil {
		return nil, "", nil, err
	}
	return payload, id, normalized, nil
}

func marshalNormalizedProvider(provider *providerv1.Provider) ([]byte, string, error) {
	id := strings.TrimSpace(provider.GetProviderId())
	if id == "" {
		return nil, "", domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	payload, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(provider)
	if err != nil {
		return nil, "", fmt.Errorf("platformk8s/providers: marshal provider %q: %w", id, err)
	}
	return payload, id, nil
}

func unmarshalProvider(id string, payload []byte) (*providerv1.Provider, error) {
	provider := &providerv1.Provider{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, provider); err != nil {
		return nil, fmt.Errorf("platformk8s/providers: unmarshal provider %q: %w", id, err)
	}
	return normalizeProviderForRead(id, provider)
}

func normalizeProviderForWrite(id string, input *providerv1.Provider) (*providerv1.Provider, error) {
	provider, err := normalizeProviderForRead(id, input)
	if err != nil {
		return nil, err
	}
	if err := providerv1.ValidateProvider(provider); err != nil {
		return nil, domainerror.NewValidation("platformk8s/providers: invalid provider %q: %v", provider.GetProviderId(), err)
	}
	return provider, nil
}

func normalizeProviderForRead(id string, input *providerv1.Provider) (*providerv1.Provider, error) {
	if input == nil {
		return nil, domainerror.NewValidation("platformk8s/providers: provider is nil")
	}
	provider := cloneProviderProto(input)
	id = strings.TrimSpace(id)
	if strings.TrimSpace(provider.GetProviderId()) == "" {
		provider.ProviderId = id
	}
	provider.ProviderId = strings.TrimSpace(provider.GetProviderId())
	if provider.GetProviderId() == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	if id != "" && provider.GetProviderId() != id {
		return nil, domainerror.NewValidation("platformk8s/providers: provider id %q does not match stored id %q", provider.GetProviderId(), id)
	}
	return provider, nil
}

func providerNotFound(providerID string) error {
	return domainerror.NewNotFound("platformk8s/providers: provider %q not found", providerID)
}

func cloneProviderProto(input *providerv1.Provider) *providerv1.Provider {
	if input == nil {
		return &providerv1.Provider{}
	}
	return proto.Clone(input).(*providerv1.Provider)
}
