package providerobservability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/provideridentity"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
)

type SurfaceObservabilityRunner struct {
	collectors map[string]ObservabilityCollector
	metrics    *observabilityMetrics
	now        func() time.Time
	logger     *slog.Logger
	providers  providers.Store
	surfaces   SurfaceReader
	auth       ObservabilityAuthClient
}

// Ensure interface compliance
var _ Capability = (*SurfaceObservabilityRunner)(nil)

type SurfaceObservabilityRunnerConfig struct {
	Providers  providers.Store
	Surfaces   SurfaceReader
	Collectors []ObservabilityCollector
	Auth       ObservabilityAuthClient
	Logger     *slog.Logger
	Now        func() time.Time
}

type SurfaceReader interface {
	Get(context.Context, string) (*supportv1.Surface, error)
}

func NewSurfaceObservabilityRunner(config SurfaceObservabilityRunnerConfig) (*SurfaceObservabilityRunner, error) {
	switch {
	case config.Providers == nil:
		return nil, fmt.Errorf("providerobservability: providers store is nil")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
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
		collectors: collectors,
		metrics:    metrics,
		now:        config.Now,
		logger:     config.Logger,
		providers:  config.Providers,
		surfaces:   config.Surfaces,
		auth:       config.Auth,
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
	if r.surfaces == nil {
		return "", false
	}
	surface, err := r.surfaces.Get(ctx, surfaceID)
	if err != nil {
		return "", false
	}
	schemaID = strings.TrimSpace(surface.GetQuotaProbeId())
	if schemaID == "" {
		return "", false
	}
	_, ok = r.collectors[schemaID]
	return schemaID, ok
}

func (r *SurfaceObservabilityRunner) ProbeProvider(ctx context.Context, target ProbeTarget, _ Trigger) (*ProbeResult, error) {
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
		return stampProbeResult(result, r.now().UTC()), nil
	}
	return r.probeProvider(ctx, provider, schemaID)
}

func (r *SurfaceObservabilityRunner) probeProvider(ctx context.Context, provider *providerv1.Provider, schemaID string) (*ProbeResult, error) {
	providerID := strings.TrimSpace(provider.GetProviderId())
	now := r.now().UTC()
	result := &ProbeResult{
		OwnerKind:  OwnerKindSurface,
		SchemaID:   schemaID,
		ProviderID: providerID,
		SurfaceID:  strings.TrimSpace(provider.GetSurfaceId()),
	}

	collector, ok := r.collectors[schemaID]
	if !ok {
		result.Outcome = ProbeOutcomeUnsupported
		result.Message = "unsupported observability surface schema"
		return stampProbeResult(result, now), nil
	}

	httpClient, err := observabilityHTTPClient(ctx)
	if err != nil {
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(err)
		result.Message = err.Error()
		return stampProbeResult(result, now), nil
	}

	// Prepare input
	input := ObservabilityCollectInput{
		ProviderID:   providerID,
		SurfaceID:    strings.TrimSpace(provider.GetSurfaceId()),
		CredentialID: provideridentity.ObservabilityCredentialID(providerID),
		Auth:         r.auth,
		SchemaID:     schemaID,
		HTTPClient:   httpClient,
	}

	collectResult, collectErr := collector.Collect(ctx, input)

	if collectErr != nil {
		if isObservabilityUnauthorizedError(collectErr) {
			result.Outcome = ProbeOutcomeAuthBlocked
			result.Reason = observabilityUnauthorizedReason(collectErr)
			result.Message = observabilityUnauthorizedSafeMessage(collectErr)
			return stampProbeResult(result, now), nil
		}
		result.Outcome = ProbeOutcomeFailed
		result.Reason = observabilityFailureReasonFromError(collectErr)
		result.Message = "observability probe failed"
		return stampProbeResult(result, now), nil
	}

	if collectResult != nil {
		r.metrics.recordCollectorValues(schemaID, providerID, collectResult.GaugeRows)
	}

	result.Outcome = ProbeOutcomeExecuted
	result.Message = "observability probe succeeded"
	return stampProbeResult(result, now), nil
}

func stampProbeResult(result *ProbeResult, now time.Time) *ProbeResult {
	if result == nil {
		return nil
	}
	result.LastAttemptAt = timePointerCopy(&now)
	return result
}
