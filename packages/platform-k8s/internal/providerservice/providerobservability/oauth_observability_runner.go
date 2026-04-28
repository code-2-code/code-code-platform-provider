package providerobservability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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

type OAuthObservabilityRunner struct {
	probeStateTracker
	credentialFreshener OAuthCredentialFreshener
	credentialReader    CredentialMaterialReader
	credentialMerger    CredentialMaterialValueMerger
	cliSupport          *clisupport.ManagementService
	collectors          map[string]ObservabilityCollector
	now                 func() time.Time
	logger              *slog.Logger
	providers           providers.Store
	cliVersions         cliversions.Store
}

type OAuthObservabilityRunnerConfig struct {
	Providers           providers.Store
	CLIVersions         cliversions.Store
	CredentialFreshener OAuthCredentialFreshener
	CredentialReader    CredentialMaterialReader
	CredentialMerger    CredentialMaterialValueMerger
	Collectors          []ObservabilityCollector
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
	metrics, err := newObservabilityMetrics("gen_ai.provider.cli.oauth", "cli_id")
	if err != nil {
		return nil, err
	}
	collectorList := config.Collectors
	if len(collectorList) == 0 {
		collectorList = DefaultOAuthCollectors()
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
	return &OAuthObservabilityRunner{
		probeStateTracker:   newProbeStateTracker(metrics, oauthObservabilityFailureBackoff),
		credentialFreshener: config.CredentialFreshener,
		credentialReader:    config.CredentialReader,
		credentialMerger:    config.CredentialMerger,
		cliSupport:          cliSupport,
		collectors:          collectors,
		now:                 config.Now,
		logger:              config.Logger,
		providers:           config.Providers,
		cliVersions:         config.CLIVersions,
	}, nil
}

func (r *OAuthObservabilityRunner) ProbeProvider(ctx context.Context, providerID string, trigger Trigger) (*ProbeResult, error) {
	trimmedID := strings.TrimSpace(providerID)
	if trimmedID == "" {
		return nil, fmt.Errorf("providerobservability: oauth observability provider id is empty")
	}
	surface, err := findProbeSurface(ctx, r.providers, trimmedID, "oauth", oauthProviderProbeSurfaceSupported)
	if err != nil {
		return nil, err
	}
	if surface == nil {
		result := &ProbeResult{
			ProviderID: trimmedID,
			Outcome:    ProbeOutcomeUnsupported,
			Message:    "provider has no supported oauth observability surface",
		}
		return r.recordProbeResult(result, trigger, "", r.now().UTC(), oauthObservabilityFailureBackoff), nil
	}
	return r.probeProvider(ctx, trimmedID, surface, trigger)
}

func (r *OAuthObservabilityRunner) ProbeAllDue(ctx context.Context, trigger Trigger) error {
	targets, err := collectDueTargets(ctx, r.providers, oauthProviderProbeSurfaceSupported)
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
			r.logger.Warn("oauth observability due operation failed",
				"provider_id", providerID,
				"error", probeErr,
			)
		}
	}
	return nil
}

func oauthProviderProbeSurfaceSupported(surface *providerv1.ProviderSurfaceBinding) bool {
	if surface == nil || surface.GetRuntime() == nil {
		return false
	}
	runtime := surface.GetRuntime()
	return providerv1.RuntimeKind(runtime) == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI &&
		strings.TrimSpace(providerv1.RuntimeCLIID(runtime)) != ""
}

