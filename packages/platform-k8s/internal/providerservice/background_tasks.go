package providerservice

import (
	"context"
	"errors"
	"fmt"

	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
)

func (s *Server) runProviderCatalogDiscovery(ctx context.Context, providerIDs []string) error {
	if s == nil || s.catalogDiscovery == nil {
		return fmt.Errorf("platformk8s/providerservice: provider catalog discovery is not initialized")
	}
	return s.catalogDiscovery.Sync(ctx, providerIDs)
}

func (s *Server) DiscoverProviderCatalogs(ctx context.Context, providerIDs []string) error {
	return s.runProviderCatalogDiscovery(ctx, normalizedProviderIDs(providerIDs))
}

func (s *Server) runProviderObservabilityProbe(ctx context.Context, providerIDs []string, trigger providerobservability.Trigger) error {
	ids := append([]string(nil), providerIDs...)
	if s == nil || s.providerObservability == nil {
		return fmt.Errorf("platformk8s/providerservice: provider observability is not initialized")
	}
	probeIDs, err := s.readyProviderObservabilityTargets(ids)
	if err != nil {
		return err
	}
	if len(probeIDs) == 0 {
		return nil
	}
	var errs []error
	for _, providerID := range probeIDs {
		_, err := s.providerObservability.ProbeProvider(ctx, providerID, trigger)
		if err != nil {
			errs = append(errs, fmt.Errorf("probe provider %q: %w", providerID, err))
		}
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
	}
	return errors.Join(errs...)
}

func (s *Server) readyProviderObservabilityTargets(providerIDs []string) ([]string, error) {
	if len(providerIDs) == 0 {
		return nil, nil
	}
	return append([]string(nil), providerIDs...), nil
}
