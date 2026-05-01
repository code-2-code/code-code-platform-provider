package providerservice

import (
	"code-code.internal/platform-k8s/internal/platform/outboundhttp"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
	"code-code.internal/platform-k8s/internal/providerservice/providercatalogs"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"code-code.internal/platform-k8s/internal/providerservice/templates"
)

func assembleServer(config Config) (*Server, error) {
	surfaceMetadata, err := providersurfaces.NewService()
	if err != nil {
		return nil, err
	}
	providerRepository, err := providers.NewProviderRepository(config.StatePool)
	if err != nil {
		return nil, err
	}

	templateService, err := templates.NewTemplateManagementService(config.Client, providerRepository)
	if err != nil {
		return nil, err
	}
	providerAccounts, err := providers.NewService(providers.Config{
		StatePool: config.StatePool,
		Surfaces:  surfaceMetadata,
	})
	if err != nil {
		return nil, err
	}
	probeHTTPClientFactory := outboundhttp.NewClientFactory()
	probeExecutor, err := providercatalogs.NewCatalogProbeExecutor(
		probeHTTPClientFactory,
		config.Client,
		config.Namespace,
	)
	if err != nil {
		return nil, err
	}
	catalogMaterializer := providercatalogs.NewCatalogMaterializer(
		providercatalogs.NewMaterializerProbe(probeExecutor, nil),
		config.Logger,
		providerCatalogModelFilter,
		surfaceMetadata,
	)
	collectors := append(providerobservability.DefaultVendorCollectors(), providerobservability.DefaultOAuthCollectors()...)
	surfaceObservability, err := providerobservability.NewSurfaceObservabilityRunner(providerobservability.SurfaceObservabilityRunnerConfig{
		Providers:  providerRepository,
		Surfaces:   surfaceMetadata,
		Collectors: collectors,
		Logger:     config.Logger,
	})
	if err != nil {
		return nil, err
	}
	providerObservability, err := providerobservability.NewService(providerobservability.Config{
		Providers: providerRepository,
		Capabilities: []providerobservability.Capability{
			surfaceObservability,
		},
	})
	if err != nil {
		return nil, err
	}
	return &Server{
		surfaceMetadata:         surfaceMetadata,
		providers:               providerAccounts,
		templates:               templateService,
		providerObservability:   providerObservability,
		catalogDiscovery:        providercatalogs.NewMaterializationSyncer(providerRepository, catalogMaterializer, config.Logger),
		providerHostTargetLimit: config.ProviderHostTelemetryMaxTargets,
	}, nil
}
