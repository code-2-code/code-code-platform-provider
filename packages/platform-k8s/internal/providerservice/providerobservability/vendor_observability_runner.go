package providerobservability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
)

const (
	vendorObservabilityPendingBackoff = time.Minute
	vendorObservabilityFailureBackoff = 5 * time.Minute
)

type VendorObservabilityProbeTrigger string

const (
	VendorObservabilityProbeTriggerSchedule VendorObservabilityProbeTrigger = "schedule"
	VendorObservabilityProbeTriggerManual   VendorObservabilityProbeTrigger = "manual"
	VendorObservabilityProbeTriggerConnect  VendorObservabilityProbeTrigger = "connect"
)

type VendorObservabilityProbeOutcome string

const (
	VendorObservabilityProbeOutcomeExecuted    VendorObservabilityProbeOutcome = "executed"
	VendorObservabilityProbeOutcomeThrottled   VendorObservabilityProbeOutcome = "throttled"
	VendorObservabilityProbeOutcomeAuthBlocked VendorObservabilityProbeOutcome = "auth_blocked"
	VendorObservabilityProbeOutcomeUnsupported VendorObservabilityProbeOutcome = "unsupported"
	VendorObservabilityProbeOutcomeFailed      VendorObservabilityProbeOutcome = "failed"
)

type VendorObservabilityProbeResult struct {
	VendorID                 string
	ProviderID               string
	ProviderSurfaceBindingID string
	Outcome                  VendorObservabilityProbeOutcome
	Message                  string
	Reason                   string
	LastAttemptAt            *time.Time
	NextAllowedAt            *time.Time
}

type vendorObservabilityState struct {
	lastAttemptAt time.Time
	nextAllowedAt time.Time
}

type VendorObservabilityRunner struct {
	vendorSupport    *vendorsupport.ManagementService
	credentialReader CredentialMaterialReader
	credentialMerger CredentialMaterialValueMerger
	collectors       map[string]VendorObservabilityCollector
	now              func() time.Time
	logger           *slog.Logger
	metrics          *vendorObservabilityMetrics
	providers        providers.Store

	mu     sync.Mutex
	states map[string]vendorObservabilityState
}

type VendorObservabilityRunnerConfig struct {
	Providers        providers.Store
	CredentialReader CredentialMaterialReader
	CredentialMerger CredentialMaterialValueMerger
	Collectors       []VendorObservabilityCollector
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
	metrics, err := registerVendorObservabilityMetrics()
	if err != nil {
		return nil, err
	}
	collectorList := config.Collectors
	if len(collectorList) == 0 {
		collectorList = DefaultVendorObservabilityCollectors()
	}
	collectors := map[string]VendorObservabilityCollector{}
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
		vendorSupport:    vendorSupport,
		credentialReader: config.CredentialReader,
		credentialMerger: config.CredentialMerger,
		collectors:       collectors,
		now:              config.Now,
		logger:           config.Logger,
		metrics:          metrics,
		providers:        config.Providers,
		states:           map[string]vendorObservabilityState{},
	}, nil
}

func (r *VendorObservabilityRunner) ProbeProvider(ctx context.Context, providerID string, trigger VendorObservabilityProbeTrigger) (*VendorObservabilityProbeResult, error) {
	trimmedID := strings.TrimSpace(providerID)
	if trimmedID == "" {
		return nil, fmt.Errorf("providerobservability: vendor observability provider id is empty")
	}
	surface, err := r.providerProbeSurface(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	if surface == nil {
		result := &VendorObservabilityProbeResult{
			ProviderID: trimmedID,
			Outcome:    VendorObservabilityProbeOutcomeUnsupported,
			Message:    "provider has no supported vendor observability surface",
		}
		return r.recordProbeResult(result, trigger, r.now().UTC(), vendorObservabilityFailureBackoff), nil
	}
	return r.probeProvider(ctx, trimmedID, surface, trigger)
}

func (r *VendorObservabilityRunner) ProbeAllDue(ctx context.Context, trigger VendorObservabilityProbeTrigger) error {
	items, err := providers.ListSurfaceBindingProjections(ctx, r.providers)
	if err != nil {
		return err
	}
	targets := map[string]*providerv1.ProviderSurfaceBinding{}
	for _, item := range items {
		providerID := vendorProjectionProviderID(&item)
		if providerID == "" || targets[providerID] != nil {
			continue
		}
		if vendorProviderProbeSurfaceSupported(item.Surface) {
			targets[providerID] = item.Surface
		}
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

func (r *VendorObservabilityRunner) providerProbeSurface(ctx context.Context, providerID string) (*providerv1.ProviderSurfaceBinding, error) {
	items, err := providers.ListSurfaceBindingProjections(ctx, r.providers)
	if err != nil {
		return nil, err
	}
	found := false
	for _, item := range items {
		if vendorProjectionProviderID(&item) != providerID {
			continue
		}
		found = true
		if vendorProviderProbeSurfaceSupported(item.Surface) {
			return item.Surface, nil
		}
	}
	if !found {
		return nil, domainerror.NewNotFound("providerobservability: vendor observability provider %q not found", providerID)
	}
	return nil, nil
}

func vendorProjectionProviderID(item *providers.SurfaceBindingProjection) string {
	if item == nil {
		return ""
	}
	if id := strings.TrimSpace(item.Provider.GetProviderId()); id != "" {
		return id
	}
	return ""
}

func vendorProviderProbeSurfaceSupported(surface *providerv1.ProviderSurfaceBinding) bool {
	if surface == nil || surface.GetRuntime() == nil {
		return false
	}
	return providerv1.RuntimeKind(surface.GetRuntime()) == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API &&
		vendorOwnerID(surface) != ""
}
