package providers

import (
	"fmt"
	"slices"
	"sync"

	providercontract "code-code.internal/agent-runtime-contract/provider"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

// Lookup stores provider implementations keyed by surface id.
type Lookup struct {
	mu        sync.RWMutex
	providers map[string]providercontract.Provider
}

// NewLookup creates an empty provider implementation lookup.
func NewLookup() *Lookup {
	return &Lookup{
		providers: map[string]providercontract.Provider{},
	}
}

// Register adds one provider implementation keyed by Surface().SurfaceId.
func (l *Lookup) Register(provider providercontract.Provider) error {
	if provider == nil {
		return fmt.Errorf("platformk8s/providers: provider is nil")
	}
	surface := provider.Surface()
	if err := providerv1.ValidateProviderSurface(surface); err != nil {
		return fmt.Errorf("platformk8s/providers: invalid provider surface: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.providers[surface.GetSurfaceId()]; exists {
		return fmt.Errorf("platformk8s/providers: provider surface %q is already registered", surface.GetSurfaceId())
	}
	l.providers[surface.GetSurfaceId()] = provider
	return nil
}

// Get returns one provider implementation by surface id.
func (l *Lookup) Get(surfaceID string) (providercontract.Provider, error) {
	if surfaceID == "" {
		return nil, fmt.Errorf("platformk8s/providers: surface id is empty")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	provider, ok := l.providers[surfaceID]
	if !ok {
		return nil, fmt.Errorf("platformk8s/providers: provider surface %q is not registered", surfaceID)
	}
	return provider, nil
}

// GetSurface returns one provider surface by stable identity.
func (l *Lookup) GetSurface(surfaceID string) (*providerv1.ProviderSurface, error) {
	provider, err := l.Get(surfaceID)
	if err != nil {
		return nil, err
	}
	return proto.Clone(provider.Surface()).(*providerv1.ProviderSurface), nil
}

// ListSurfaces returns all registered provider surfaces sorted by id.
func (l *Lookup) ListSurfaces() []*providerv1.ProviderSurface {
	l.mu.RLock()
	defer l.mu.RUnlock()

	items := make([]*providerv1.ProviderSurface, 0, len(l.providers))
	for _, provider := range l.providers {
		items = append(items, proto.Clone(provider.Surface()).(*providerv1.ProviderSurface))
	}
	slices.SortFunc(items, func(left, right *providerv1.ProviderSurface) int {
		switch {
		case left.GetSurfaceId() < right.GetSurfaceId():
			return -1
		case left.GetSurfaceId() > right.GetSurfaceId():
			return 1
		default:
			return 0
		}
	})
	return items
}
