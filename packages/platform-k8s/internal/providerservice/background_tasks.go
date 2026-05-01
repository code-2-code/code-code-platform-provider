package providerservice

import (
	"context"
	"errors"
	"fmt"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
)

func (s *Server) runProviderCatalogDiscovery(ctx context.Context, providerIDs []string) error {
	if s == nil || s.catalogDiscovery == nil {
		return fmt.Errorf("platformk8s/providerservice: provider catalog discovery is not initialized")
	}
	return s.catalogDiscovery.Sync(ctx, providerIDs)
}

func (s *Server) runProviderObservabilityProbe(ctx context.Context, providerIDs []string, trigger providerobservability.Trigger) ([]*managementv1.ProbeProviderObservabilityResponse, error) {
	ids := append([]string(nil), providerIDs...)
	if s == nil || s.providerObservability == nil {
		return nil, fmt.Errorf("platformk8s/providerservice: provider observability is not initialized")
	}
	probeIDs, err := s.readyProviderObservabilityTargets(ids)
	if err != nil {
		return nil, err
	}
	if len(probeIDs) == 0 {
		return nil, nil
	}
	var errs []error
	results := make([]*managementv1.ProbeProviderObservabilityResponse, 0, len(probeIDs))
	for _, providerID := range probeIDs {
		result, err := s.providerObservability.ProbeProvider(ctx, providerID, trigger)
		if err != nil {
			errs = append(errs, fmt.Errorf("probe provider %q: %w", providerID, err))
		} else if result != nil {
			results = append(results, result)
		}
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
	}
	return results, errors.Join(errs...)
}

func (s *Server) readyProviderObservabilityTargets(providerIDs []string) ([]string, error) {
	if len(providerIDs) == 0 {
		return nil, nil
	}
	return append([]string(nil), providerIDs...), nil
}
