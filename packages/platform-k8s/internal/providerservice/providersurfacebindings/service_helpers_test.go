package providersurfacebindings

import (
	"context"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type memoryProviderStore struct {
	items map[string]*providerv1.Provider
}

func newServiceForTest(providers ...*providerv1.Provider) (*Service, *memoryProviderStore) {
	store := &memoryProviderStore{items: map[string]*providerv1.Provider{}}
	for _, provider := range providers {
		_, _ = store.Upsert(context.Background(), provider)
	}
	return &Service{repository: &Repository{providers: store}}, store
}

func (s *memoryProviderStore) List(context.Context) ([]*providerv1.Provider, error) {
	items := make([]*providerv1.Provider, 0, len(s.items))
	for _, provider := range s.items {
		items = append(items, proto.Clone(provider).(*providerv1.Provider))
	}
	return items, nil
}

func (s *memoryProviderStore) Get(_ context.Context, providerID string) (*providerv1.Provider, error) {
	provider := s.items[strings.TrimSpace(providerID)]
	if provider == nil {
		return nil, domainerror.NewNotFound("provider %q not found", providerID)
	}
	return proto.Clone(provider).(*providerv1.Provider), nil
}

func (s *memoryProviderStore) Upsert(_ context.Context, provider *providerv1.Provider) (*providerv1.Provider, error) {
	if provider == nil || strings.TrimSpace(provider.GetProviderId()) == "" {
		return nil, fmt.Errorf("provider id is empty")
	}
	clone := proto.Clone(provider).(*providerv1.Provider)
	s.items[clone.GetProviderId()] = clone
	return proto.Clone(clone).(*providerv1.Provider), nil
}

func (s *memoryProviderStore) Update(ctx context.Context, providerID string, mutate func(*providerv1.Provider) error) (*providerv1.Provider, error) {
	provider, err := s.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if err := mutate(provider); err != nil {
		return nil, err
	}
	return s.Upsert(ctx, provider)
}

func (s *memoryProviderStore) Delete(_ context.Context, providerID string) error {
	providerID = strings.TrimSpace(providerID)
	if s.items[providerID] == nil {
		return domainerror.NewNotFound("provider %q not found", providerID)
	}
	delete(s.items, providerID)
	return nil
}
