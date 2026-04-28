package providerservice

import (
	"regexp"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
	"code-code.internal/platform-k8s/internal/cliruntimeservice/cliversions"
	"code-code.internal/platform-k8s/internal/platform/outboundhttp"
	"code-code.internal/platform-k8s/internal/providerservice/providercatalogs"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"code-code.internal/platform-k8s/internal/providerservice/providersurfacebindings"
	cliidentity "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/identity"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
	"code-code.internal/platform-k8s/internal/supportservice/providersurfaces"
	"code-code.internal/platform-k8s/internal/supportservice/templates"
	vendoridentity "code-code.internal/platform-k8s/internal/supportservice/vendors/identity"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
)

// defaultCatalogExcludePattern excludes non-text-generation provider models
// (image gen, audio, video, embedding, robotics, moderation, etc.) from the
// materialized catalog. Adjust this pattern to change which model IDs are filtered.
var defaultCatalogExcludePattern = regexp.MustCompile(
	`(?i)(^|[^a-z0-9])` +
		`(imagen|veo|lyria|tts|stt` +
		`|audio|speech|voice` +
		`|robotics|computer-use` +
		`|embedding|embed` +
		`|moderation|safety` +
		`|deep-research` +
		`|nano-banana` +
		`)` +
		`([^a-z0-9]|$)`,
)

func assembleServer(config Config) (*Server, error) {
	credentialService := newAuthCredentialService(authv1.NewAuthServiceClient(config.AuthConn))
	providerSurfaceBindings, err := providersurfacebindings.NewService(config.StatePool)
	if err != nil {
		return nil, err
	}
	vendors, err := vendoridentity.NewVendorManagementService()
	if err != nil {
		return nil, err
	}
	cliDefinitions, err := cliidentity.NewCLIDefinitionManagementService()
	if err != nil {
		return nil, err
	}
	vendorSupport, err := vendorsupport.NewManagementService()
	if err != nil {
		return nil, err
	}
	cliSupport, err := clisupport.NewManagementService()
	if err != nil {
		return nil, err
	}
	surfaceMetadata, err := providersurfaces.NewService(cliSupport, vendorSupport)
	if err != nil {
		return nil, err
	}
	providerRepository, err := providers.NewProviderRepository(config.StatePool)
	if err != nil {
		return nil, err
	}
	cliVersions, err := cliversions.NewPostgresStore(config.StatePool)
	if err != nil {
		return nil, err
	}
	templateService, err := templates.NewTemplateManagementService(config.Client, providerRepository)
	if err != nil {
		return nil, err
	}
	providerAccounts, err := providers.NewService(providers.Config{
		StatePool:   config.StatePool,
		Credentials: credentialService,
		Vendors:     vendors,
		CLIDefs:     cliSupportIconReferenceService{source: cliSupport},
		CLISupport:  cliSupport,
	})
	if err != nil {
		return nil, err
	}
	postConnect := config.PostConnect
	authClient := authv1.NewAuthServiceClient(config.AuthConn)
	probeExecutor, err := providercatalogs.NewCatalogProbeExecutor(outboundhttp.NewClientFactory(), config.Client, config.Namespace)
	if err != nil {
		return nil, err
	}
	catalogMaterializer := providercatalogs.NewCatalogMaterializer(
		providercatalogs.NewMaterializerProbe(probeExecutor),
		config.Logger,
		providercatalogs.ExcludeByPattern(defaultCatalogExcludePattern),
	)
	providerConnect, err := providerconnect.NewService(providerconnect.Config{
		Client:         config.Client,
		Reader:         config.Reader,
		Namespace:      config.Namespace,
		Credentials:    credentialService,
		Providers:      providerConnectSurfaceBindingAdapter{source: providerSurfaceBindings},
		ProviderReader: providerConnectProviderAdapter{source: providerAccounts},
		Surfaces:       surfaceMetadata,
		VendorSupport:  vendorSupport,
		CLISupport:     cliSupport,
		PostConnect:    postConnect,
		OAuthSessions:  newRemoteOAuthSessionService(oauthv1.NewOAuthSessionServiceClient(config.AuthConn)),
		Logger:         config.Logger,
	})
	if err != nil {
		return nil, err
	}
	oauthObservability, err := providerobservability.NewOAuthObservabilityRunner(providerobservability.OAuthObservabilityRunnerConfig{
		Providers:           providerRepository,
		CLIVersions:         cliVersions,
		CredentialFreshener: credentialService,
		CredentialReader:    credentialService,
		CredentialMerger:    credentialService,
		Logger:              config.Logger,
	})
	if err != nil {
		return nil, err
	}
	vendorObservability, err := providerobservability.NewVendorObservabilityRunner(providerobservability.VendorObservabilityRunnerConfig{
		Providers:        providerRepository,
		CredentialReader: credentialService,
		CredentialMerger: credentialService,
		Logger:           config.Logger,
	})
	if err != nil {
		return nil, err
	}
	providerObservability, err := providerobservability.NewService(providerobservability.Config{
		ProviderSurfaceBindings: providerSurfaceBindings,
		Capabilities: []providerobservability.Capability{
			oauthObservabilityCapability{runner: oauthObservability},
			vendorObservabilityCapability{runner: vendorObservability},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Server{
		surfaceMetadata:         surfaceMetadata,
		authClient:              authClient,
		providers:               providerAccounts,
		providerSurfaceBindings: providerSurfaceBindings,
		providerConnect:         providerConnect,
		vendors:                 vendors,
		vendorSupport:           vendorSupport,
		cliDefinitions:          cliDefinitions,
		templates:               templateService,
		providerObservability:   providerObservability,
		catalogDiscovery:        providercatalogs.NewMaterializationSyncer(providerRepository, catalogMaterializer, config.Logger),
		providerHostTargetLimit: config.ProviderHostTelemetryMaxTargets,
	}, nil
}
