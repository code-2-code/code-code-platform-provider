package providercatalogs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	providerv1 "code-code.internal/go-contract/provider/v1"
	provideraggregates "code-code.internal/platform-k8s/internal/providerservice/providers"
	"google.golang.org/protobuf/proto"
)

// MaterializationSyncer refreshes provider surface catalogs from model catalog sources.
type MaterializationSyncer struct {
	providers    provideraggregates.Store
	materializer *CatalogMaterializer
	logger       *slog.Logger
}

func NewMaterializationSyncer(
	providers provideraggregates.Store,
	materializer *CatalogMaterializer,
	logger *slog.Logger,
) *MaterializationSyncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &MaterializationSyncer{
		providers:    providers,
		materializer: materializer,
		logger:       logger,
	}
}

func (s *MaterializationSyncer) Sync(ctx context.Context, providerIDs []string) error {
	if s == nil || s.providers == nil || s.materializer == nil {
		return nil
	}
	ids, err := s.providerIDs(ctx, providerIDs)
	if err != nil {
		return err
	}
	for _, providerID := range ids {
		if err := s.syncProvider(ctx, providerID); err != nil {
			return err
		}
	}
	return nil
}

func (s *MaterializationSyncer) providerIDs(ctx context.Context, values []string) ([]string, error) {
	ids := normalizeProviderIDs(values)
	if len(ids) > 0 {
		return ids, nil
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	ids = make([]string, 0, len(providers))
	for _, provider := range providers {
		if id := strings.TrimSpace(provider.GetProviderId()); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *MaterializationSyncer) syncProvider(ctx context.Context, providerID string) error {
	_, err := s.providers.Update(ctx, providerID, func(next *providerv1.Provider) error {
		materialized, err := s.materializer.MaterializeProvider(ctx, next)
		if err != nil {
			return err
		}
		proto.Reset(next)
		proto.Merge(next, materialized)
		return nil
	})
	if err != nil {
		return fmt.Errorf("platformk8s/providercatalogs: materialize provider %q: %w", providerID, err)
	}
	return nil
}

func normalizeProviderIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
