package providerobservability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
)

const (
	surfaceObservabilityPendingBackoff = time.Minute
	surfaceObservabilityFailureBackoff = 5 * time.Minute
)

type SurfaceObservabilityRunner struct {
	probeStateTracker
	collectors      map[string]ObservabilityCollector
	now             func() time.Time
	logger          *slog.Logger
	providers       providers.Store
	surfaceRegistry SurfaceRegistry
}

// Ensure interface compliance
var _ Capability = (*SurfaceObservabilityRunner)(nil)

type SurfaceRegistry interface {
	Get(ctx context.Context, surfaceID string) (*providerv1.ProviderSurface, error)
}

type SurfaceObservabilityRunnerConfig struct {
	Providers       providers.Store
	SurfaceRegistry SurfaceRegistry
	Collectors      []ObservabilityCollector
	Logger          *slog.Logger
	Now             func() time.Time
}

func NewSurfaceObservabilityRunner(config SurfaceObservabilityRunnerConfig) (*SurfaceObservabilityRunner, error) {
	switch {
	case config.Providers == nil:
		return nil, fmt.Errorf("providerobservability: providers store is nil")
	case config.SurfaceRegistry == nil:
		return nil, fmt.Errorf("providerobservability: surface registry is nil")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	metrics, err := newObservabilityMetrics("gen_ai.provider.surface.probe", "schema_id")
	if err != nil {
		return nil, err
	}
	
	collectors := map[string]ObservabilityCollector{}
	for _, collector := range config.Collectors {
		if collector == nil {
			continue
		}
		collectorID := strings.TrimSpace(collector.CollectorID())
		if collectorID == "" {
			continue
		}
		collectors[collectorID] = collector
	}
	return &SurfaceObservabilityRunner{
		probeStateTracker: newProbeStateTracker(metrics, surfaceObservabilityFailureBackoff),
		collectors:        collectors,
		now:               config.Now,
		logger:            config.Logger,
		providers:         config.Providers,
		surfaceRegistry:   config.SurfaceRegistry,
	}, nil
}

func (r *SurfaceObservabilityRunner) OwnerKind() OwnerKind {
	return OwnerKindSurface
}

func (r *SurfaceObservabilityRunner) Supports(ctx context.Context, provider *providerv1.Provider) (schemaID string, ok bool) {
	if provider == nil {
		return "", false
	}
	surfaceID := strings.TrimSpace(provider.GetSurfaceId())
	if surfaceID == "" {
		return "", false
	}
	surface, err := r.surfaceRegistry.Get(ctx, surfaceID)
	if err != nil || surface == nil || surface.GetProbes() == nil || surface.GetProbes().GetQuota() == nil {
		return "", false
	}
	schemaID = strings.TrimSpace(surface.GetProbes().GetQuota().GetSchemaId())
	if schemaID == "" {
		return "", false
	}
	_, ok = r.collectors[schemaID]
	return schemaID, ok
}

func (r *SurfaceObservabilityRunner) ProbeProvider(ctx context.Context, target ProbeTarget, trigger Trigger) (*ProbeResult, error) {
	trimmedID := strings.TrimSpace(target.ProviderID)
	if trimmedID == "" {
		return nil, fmt.Errorf("providerobservability: provider id is empty")
	}
	provider, err := r.providers.Get(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	schemaID, ok := r.Supports(ctx, provider)
	if !ok {
		result := &ProbeResult{
			ProviderID: trimmedID,
			Outcome:    ProbeOutcomeUnsupported,
			Message:    "provider has no supported observability surface",
		}
		return r.recordState(result, trigger, r.now().UTC(), surfaceObservabilityFailureBackoff), nil
	}
	return r.probeProvider(ctx, provider, schemaID, trigger)
}

func (r *SurfaceObservabilityRunner) ProbeAllDue(ctx context.Context, trigger Trigger) error {
	items, err := r.providers.List(ctx)
	if err != nil {
		return err
	}
	now := r.now().UTC()
	for _, provider := range items {
		if provider == nil {
			continue
		}
		providerID := strings.TrimSpace(provider.GetProviderId())
		if providerID == "" {
			continue
		}
		schemaID, ok := r.Supports(ctx, provider)
		if !ok {
			continue
		}
		nextAllowedAt := r.nextAllowed(providerID, schemaID)
		if !nextAllowedAt.IsZero() && now.Before(nextAllowedAt) {
			continue
		}
		if _, probeErr := r.probeProvider(ctx, provider, schemaID, trigger); probeErr != nil {
			r.logger.Warn("surface observability due operation failed",
				"provider_id", providerID,
				"error", probeErr,
			)
		}
	}
	return nil
}

func (r *SurfaceObservabilityRunner) probeProvider(ctx context.Context, provider *providerv1.Provider, schemaID string, trigger Trigger) (*ProbeResult, error) {
	providerID := strings.TrimSpace(provider.GetProviderId())
	now := r.now().UTC()
	result := &ProbeResult{
		OwnerKind:  OwnerKindSurface,
		SchemaID:   schemaID,
		ProviderID: providerID,
	}
	
	if throttled, ok := probeThrottled(&r.probeStateTracker, r.metrics, result, schemaID, trigger, now); ok {
		return throttled, nil
	}

	collector, ok := r.collectors[schemaID]
	if !ok {
		result.Outcome = ProbeOutcomeUnsupported
		result.Message = "unsupported observability surface schema"
		return r.recordState(result, trigger, now, surfaceObservabilityFailureBackoff), nil
	}

	httpClient, err := observabilityHTTPClient(ctx)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return r.recordState(result, trigger, now, surfaceObservabilityFailureBackoff), nil
	}
	
	// Prepare input
	input := ObservabilityCollectInput{
		ProviderID: providerID,
		SurfaceID:  strings.TrimSpace(provider.GetSurfaceId()),
		SchemaID:   schemaID,
		HTTPClient: httpClient,
	}

	collectResult, collectErr := collector.Collect(ctx, input)
	
	if collectErr != nil {
		if isObservabilityUnauthorizedError(collectErr) {
			result.Outcome = ProbeOutcomeAuthBlocked
			result.Reason = observabilityUnauthorizedReason(collectErr)
			result.Message = "observability credential unauthorized"
			return r.recordState(result, trigger, now, surfaceObservabilityFailureBackoff), nil
		}
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(collectErr)
		result.Message = "observability probe failed"
		return r.recordState(result, trigger, now, surfaceObservabilityFailureBackoff), nil
	}

	if collectResult != nil {
		r.metrics.recordCollectorValues(schemaID, providerID, collectResult.GaugeRows)
	}

	result.Outcome = ProbeOutcomeExecuted
	result.Message = "observability probe succeeded"
	return r.recordState(result, trigger, now, surfaceObservabilityPendingBackoff), nil
}
