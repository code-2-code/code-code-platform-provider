package providerobservability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"code-code.internal/go-contract/domainerror"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/cliruntimeservice/cliversions"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
)

const (
	oauthObservabilityPendingBackoff = time.Minute
	oauthObservabilityFailureBackoff = 5 * time.Minute
	oauthObservabilityEnsureFreshTTL = 30 * time.Second
)

type OAuthObservabilityProbeTrigger string

const (
	OAuthObservabilityProbeTriggerSchedule OAuthObservabilityProbeTrigger = "schedule"
	OAuthObservabilityProbeTriggerManual   OAuthObservabilityProbeTrigger = "manual"
	OAuthObservabilityProbeTriggerConnect  OAuthObservabilityProbeTrigger = "connect"
)

type OAuthObservabilityProbeOutcome string

const (
	OAuthObservabilityProbeOutcomeExecuted    OAuthObservabilityProbeOutcome = "executed"
	OAuthObservabilityProbeOutcomeThrottled   OAuthObservabilityProbeOutcome = "throttled"
	OAuthObservabilityProbeOutcomeAuthBlocked OAuthObservabilityProbeOutcome = "auth_blocked"
	OAuthObservabilityProbeOutcomeUnsupported OAuthObservabilityProbeOutcome = "unsupported"
	OAuthObservabilityProbeOutcomeFailed      OAuthObservabilityProbeOutcome = "failed"
)

type OAuthObservabilityProbeResult struct {
	CLIID                    string
	ProviderID               string
	ProviderSurfaceBindingID string
	Outcome                  OAuthObservabilityProbeOutcome
	Message                  string
	Reason                   string
	LastAttemptAt            *time.Time
	NextAllowedAt            *time.Time
}

type OAuthCredentialFreshener interface {
	EnsureFresh(ctx context.Context, credentialID string, minTTL time.Duration) error
}

type CredentialMaterialValueMerger interface {
	MergeCredentialMaterialValues(ctx context.Context, credentialID string, values map[string]string) error
}

type CredentialMaterialReader interface {
	ReadCredentialMaterialFields(
		ctx context.Context,
		credentialID string,
		policyRef *authv1.CredentialMaterialReadPolicyRef,
		fieldIDs []string,
	) (map[string]string, error)
}

type oauthObservabilityState struct {
	lastAttemptAt time.Time
	nextAllowedAt time.Time
}

type OAuthObservabilityRunner struct {
	credentialFreshener OAuthCredentialFreshener
	credentialReader    CredentialMaterialReader
	credentialMerger    CredentialMaterialValueMerger
	cliSupport          *clisupport.ManagementService
	collectors          map[string]OAuthObservabilityCollector
	now                 func() time.Time
	logger              *slog.Logger
	metrics             *oauthObservabilityMetrics
	providers           providers.Store
	cliVersions         cliversions.Store

	mu     sync.Mutex
	states map[string]oauthObservabilityState
}

type OAuthObservabilityRunnerConfig struct {
	Providers           providers.Store
	CLIVersions         cliversions.Store
	CredentialFreshener OAuthCredentialFreshener
	CredentialReader    CredentialMaterialReader
	CredentialMerger    CredentialMaterialValueMerger
	Collectors          []OAuthObservabilityCollector
	Logger              *slog.Logger
	Now                 func() time.Time
}

func NewOAuthObservabilityRunner(config OAuthObservabilityRunnerConfig) (*OAuthObservabilityRunner, error) {
	switch {
	case config.CredentialFreshener == nil:
		return nil, fmt.Errorf("providerobservability: oauth credential freshener is nil")
	case config.CredentialReader == nil:
		return nil, fmt.Errorf("providerobservability: credential material reader is nil")
	case config.CredentialMerger == nil:
		return nil, fmt.Errorf("providerobservability: credential material merger is nil")
	case config.Providers == nil:
		return nil, fmt.Errorf("providerobservability: oauth observability runner provider repository is nil")
	case config.CLIVersions == nil:
		return nil, fmt.Errorf("providerobservability: oauth observability runner cli version store is nil")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	cliSupport, err := clisupport.NewManagementService()
	if err != nil {
		return nil, err
	}
	metrics, err := registerOAuthObservabilityMetrics()
	if err != nil {
		return nil, err
	}
	collectorList := config.Collectors
	if len(collectorList) == 0 {
		collectorList = DefaultOAuthObservabilityCollectors()
	}
	collectors := map[string]OAuthObservabilityCollector{}
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
	return &OAuthObservabilityRunner{
		credentialFreshener: config.CredentialFreshener,
		credentialReader:    config.CredentialReader,
		credentialMerger:    config.CredentialMerger,
		cliSupport:          cliSupport,
		collectors:          collectors,
		now:                 config.Now,
		logger:              config.Logger,
		metrics:             metrics,
		providers:           config.Providers,
		cliVersions:         config.CLIVersions,
		states:              map[string]oauthObservabilityState{},
	}, nil
}

func (r *OAuthObservabilityRunner) ProbeProvider(ctx context.Context, providerID string, trigger OAuthObservabilityProbeTrigger) (*OAuthObservabilityProbeResult, error) {
	trimmedID := strings.TrimSpace(providerID)
	if trimmedID == "" {
		return nil, fmt.Errorf("providerobservability: oauth observability provider id is empty")
	}
	surface, err := r.providerProbeSurface(ctx, trimmedID)
	if err != nil {
		return nil, err
	}
	if surface == nil {
		result := &OAuthObservabilityProbeResult{
			ProviderID: trimmedID,
			Outcome:    OAuthObservabilityProbeOutcomeUnsupported,
			Message:    "provider has no supported oauth observability surface",
		}
		return r.recordProbeResult(result, trigger, "", r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	return r.probeProvider(ctx, trimmedID, surface, trigger)
}

func (r *OAuthObservabilityRunner) ProbeAllDue(ctx context.Context, trigger OAuthObservabilityProbeTrigger) error {
	items, err := providers.ListSurfaceBindingProjections(ctx, r.providers)
	if err != nil {
		return err
	}
	targets := map[string]*providerv1.ProviderSurfaceBinding{}
	for _, item := range items {
		providerID := oauthProjectionProviderID(&item)
		if providerID == "" || targets[providerID] != nil {
			continue
		}
		if oauthProviderProbeSurfaceSupported(item.Surface) {
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
			r.logger.Warn("oauth observability due operation failed",
				"provider_id", providerID,
				"error", probeErr,
			)
		}
	}
	return nil
}

func (r *OAuthObservabilityRunner) providerProbeSurface(ctx context.Context, providerID string) (*providerv1.ProviderSurfaceBinding, error) {
	items, err := providers.ListSurfaceBindingProjections(ctx, r.providers)
	if err != nil {
		return nil, err
	}
	found := false
	for _, item := range items {
		if oauthProjectionProviderID(&item) != providerID {
			continue
		}
		found = true
		if oauthProviderProbeSurfaceSupported(item.Surface) {
			return item.Surface, nil
		}
	}
	if !found {
		return nil, domainerror.NewNotFound("providerobservability: oauth observability provider %q not found", providerID)
	}
	return nil, nil
}

func oauthProjectionProviderID(item *providers.SurfaceBindingProjection) string {
	if item == nil {
		return ""
	}
	if id := strings.TrimSpace(item.Provider.GetProviderId()); id != "" {
		return id
	}
	return ""
}

func oauthProviderProbeSurfaceSupported(surface *providerv1.ProviderSurfaceBinding) bool {
	if surface == nil || surface.GetRuntime() == nil {
		return false
	}
	runtime := surface.GetRuntime()
	return providerv1.RuntimeKind(runtime) == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI &&
		strings.TrimSpace(providerv1.RuntimeCLIID(runtime)) != ""
}
