package providerobservability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
)

const (
	vendorObservabilityPendingBackoff = time.Minute
	vendorObservabilityFailureBackoff = 5 * time.Minute
)

type VendorObservabilityRunner struct {
	probeStateTracker
	vendorSupport    *vendorsupport.ManagementService
	credentialReader CredentialMaterialReader
	credentialMerger CredentialMaterialValueMerger
	collectors       map[string]ObservabilityCollector
	now              func() time.Time
	logger           *slog.Logger
	providers        providers.Store
}

type VendorObservabilityRunnerConfig struct {
	Providers        providers.Store
	CredentialReader CredentialMaterialReader
	CredentialMerger CredentialMaterialValueMerger
	Collectors       []ObservabilityCollector
	Logger           *slog.Logger
	Now              func() time.Time
}

func NewVendorObservabilityRunner(config VendorObservabilityRunnerConfig) (*VendorObservabilityRunner, error) {
	switch {
	case config.Providers == nil:
		return nil, fmt.Errorf("providerobservability: vendor observability runner provider repository is nil")
	case config.CredentialReader == nil:
		return nil, fmt.Errorf("providerobservability: credential material reader is nil")
	case config.CredentialMerger == nil:
		return nil, fmt.Errorf("providerobservability: credential material merger is nil")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	vendorSupport, err := vendorsupport.NewManagementService()
	if err != nil {
		return nil, err
	}
	metrics, err := newObservabilityMetrics("gen_ai.provider.vendor.api_key", "vendor_id")
	if err != nil {
		return nil, err
	}
	collectorList := config.Collectors
	if len(collectorList) == 0 {
		collectorList = DefaultVendorCollectors()
	}
	collectors := map[string]ObservabilityCollector{}
	for _, collector := range collectorList {
		if collector == nil {
			continue
		}
		collectorID := strings.TrimSpace(collector.CollectorID())
		if collectorID == "" {
			continue
		}
		collectors[collectorID] = collector
	}
	return &VendorObservabilityRunner{
		probeStateTracker: newProbeStateTracker(metrics, vendorObservabilityFailureBackoff),
		vendorSupport:    vendorSupport,
		credentialReader: config.CredentialReader,
		credentialMerger: config.CredentialMerger,
		collectors:       collectors,
		now:              config.Now,
		logger:           config.Logger,
		providers:        config.Providers,
	}, nil
}

func (r *VendorObservabilityRunner) ProbeProvider(ctx context.Context, providerID string, trigger Trigger) (*ProbeResult, error) {
	trimmedID := strings.TrimSpace(providerID)
	if trimmedID == "" {
		return nil, fmt.Errorf("providerobservability: vendor observability provider id is empty")
	}
	surface, err := findProbeSurface(ctx, r.providers, trimmedID, "vendor", vendorProviderProbeSurfaceSupported)
	if err != nil {
		return nil, err
	}
	if surface == nil {
		result := &ProbeResult{
			ProviderID: trimmedID,
			Outcome:    ProbeOutcomeUnsupported,
			Message:    "provider has no supported vendor observability surface",
		}
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	return r.probeProvider(ctx, trimmedID, surface, trigger)
}

func (r *VendorObservabilityRunner) ProbeAllDue(ctx context.Context, trigger Trigger) error {
	targets, err := collectDueTargets(ctx, r.providers, vendorProviderProbeSurfaceSupported)
	if err != nil {
		return err
	}
	now := r.now().UTC()
	for providerID, surface := range targets {
		nextAllowedAt := r.nextAllowed(providerID, "")
		if !nextAllowedAt.IsZero() && now.Before(nextAllowedAt) {
			continue
		}
		if _, probeErr := r.probeProvider(ctx, providerID, surface, trigger); probeErr != nil {
			r.logger.Warn("vendor observability due operation failed",
				"provider_id", providerID,
				"error", probeErr,
			)
		}
	}
	return nil
}

func vendorProviderProbeSurfaceSupported(surface *providerv1.ProviderSurfaceBinding) bool {
	if surface == nil || surface.GetRuntime() == nil {
		return false
	}
	return providerv1.RuntimeKind(surface.GetRuntime()) == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API &&
		vendorOwnerID(surface) != ""
}

