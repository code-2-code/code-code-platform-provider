package providerobservability

import (
	"context"
	"fmt"
	"testing"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestSurfaceObservabilityRunnerSupportsSurfaceQuotaProbeID(t *testing.T) {
	runner, err := NewSurfaceObservabilityRunner(SurfaceObservabilityRunnerConfig{
		Providers:  surfaceRunnerProviderStore{},
		Surfaces:   surfaceRunnerSurfaceReader{"gemini": {SurfaceId: "gemini", QuotaProbeId: "google-aistudio-quotas"}},
		Collectors: []ObservabilityCollector{surfaceRunnerCollector{id: "google-aistudio-quotas"}},
	})
	if err != nil {
		t.Fatalf("NewSurfaceObservabilityRunner() error = %v", err)
	}

	schemaID, ok := runner.Supports(context.Background(), &providerv1.Provider{
		ProviderId: "provider-google",
		SurfaceId:  "gemini",
	})
	if !ok {
		t.Fatal("Supports() ok = false, want true")
	}
	if schemaID != "google-aistudio-quotas" {
		t.Fatalf("Supports() schemaID = %q, want google-aistudio-quotas", schemaID)
	}
}

func TestSurfaceObservabilityRunnerRequiresSurfaceQuotaProbeID(t *testing.T) {
	runner, err := NewSurfaceObservabilityRunner(SurfaceObservabilityRunnerConfig{
		Providers:  surfaceRunnerProviderStore{},
		Surfaces:   surfaceRunnerSurfaceReader{"gemini": {SurfaceId: "gemini"}},
		Collectors: []ObservabilityCollector{surfaceRunnerCollector{id: "google-aistudio-quotas"}},
	})
	if err != nil {
		t.Fatalf("NewSurfaceObservabilityRunner() error = %v", err)
	}

	if schemaID, ok := runner.Supports(context.Background(), &providerv1.Provider{
		ProviderId: "provider-google",
		SurfaceId:  "gemini",
	}); ok || schemaID != "" {
		t.Fatalf("Supports() = (%q, %v), want empty false", schemaID, ok)
	}
}

type surfaceRunnerSurfaceReader map[string]*supportv1.Surface

func (r surfaceRunnerSurfaceReader) Get(_ context.Context, surfaceID string) (*supportv1.Surface, error) {
	return r[surfaceID], nil
}

type surfaceRunnerCollector struct {
	id string
}

func (c surfaceRunnerCollector) CollectorID() string {
	return c.id
}

func (surfaceRunnerCollector) Collect(context.Context, ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	return &ObservabilityCollectResult{}, nil
}

type surfaceRunnerProviderStore struct {
	items []*providerv1.Provider
}

func (s surfaceRunnerProviderStore) List(context.Context) ([]*providerv1.Provider, error) {
	return s.items, nil
}

func (s surfaceRunnerProviderStore) Get(_ context.Context, providerID string) (*providerv1.Provider, error) {
	for _, item := range s.items {
		if item.GetProviderId() == providerID {
			return item, nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", providerID)
}

func (surfaceRunnerProviderStore) Upsert(context.Context, *providerv1.Provider) (*providerv1.Provider, error) {
	return nil, fmt.Errorf("not implemented")
}

func (surfaceRunnerProviderStore) Update(context.Context, string, func(*providerv1.Provider) error) (*providerv1.Provider, error) {
	return nil, fmt.Errorf("not implemented")
}

func (surfaceRunnerProviderStore) Delete(context.Context, string) error {
	return fmt.Errorf("not implemented")
}
