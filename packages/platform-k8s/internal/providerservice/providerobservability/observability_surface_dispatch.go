package providerobservability

import (
	"context"
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
)

func projectionProviderID(item *providers.ProviderStateProjection) string {
	if item == nil {
		return ""
	}
	return strings.TrimSpace(item.Provider.GetProviderId())
}

// surfaceFilter decides whether a surface is eligible for probing.
type surfaceFilter func(provider *providerv1.Provider) bool

// findProbeSurface locates the supported surface for a given provider ID.
// Returns (nil, nil) if the provider exists but has no supported surface.
// Returns a NotFound error if the provider does not exist at all.
func findProbeSurface(
	ctx context.Context,
	store providers.Store,
	providerID string,
	kind string,
	filter surfaceFilter,
) (*providerv1.Provider, error) {
	items, err := providers.ListProviderProjections(ctx, store)
	if err != nil {
		return nil, err
	}
	found := false
	for _, item := range items {
		if projectionProviderID(&item) != providerID {
			continue
		}
		found = true
		if filter(item.Provider) {
			return item.Provider, nil
		}
	}
	if !found {
		return nil, domainerror.NewNotFound("providerobservability: %s observability provider %q not found", kind, providerID)
	}
	return nil, nil
}

// collectDueTargets builds a deduplicated map of providers matching a filter.
// Shared by both ProbeAllDue implementations.
func collectDueTargets(
	ctx context.Context,
	store providers.Store,
	filter surfaceFilter,
) (map[string]*providerv1.Provider, error) {
	items, err := providers.ListProviderProjections(ctx, store)
	if err != nil {
		return nil, err
	}
	targets := map[string]*providerv1.Provider{}
	for _, item := range items {
		providerID := projectionProviderID(&item)
		if providerID == "" || targets[providerID] != nil {
			continue
		}
		if filter(item.Provider) {
			targets[providerID] = item.Provider
		}
	}
	return targets, nil
}
