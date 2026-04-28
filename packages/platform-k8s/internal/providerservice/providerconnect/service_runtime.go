package providerconnect

import (
	"fmt"
	"log/slog"
)

type providerConnectRuntime struct {
	resources   providerConnectResources
	support     providerConnectSupport
	sessions    providerConnectSessions
	postConnect *providerConnectPostConnectWorkflow
	queries     *providerConnectQueries
	logger      *slog.Logger
}

func newProviderConnectRuntime(config Config) (providerConnectRuntime, error) {
	store, err := newSessionStore(config.Client, config.Reader, config.Namespace)
	if err != nil {
		return providerConnectRuntime{}, err
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	resources := newProviderConnectResources(config.Credentials, config.Providers)
	support := newProviderConnectSupport(config.VendorSupport, config.CLISupport)
	sessions := newProviderConnectSessions(config.OAuthSessions, store)
	queries := newProviderConnectQueries(
		config.Providers,
		config.ProviderReader,
		config.Surfaces,
	)
	return providerConnectRuntime{
		resources:   resources,
		support:     support,
		sessions:    sessions,
		postConnect: newProviderConnectPostConnectWorkflow(config.PostConnect, logger),
		queries:     queries,
		logger:      logger,
	}, nil
}

func validateProviderConnectConfig(config Config) error {
	switch {
	case config.Client == nil:
		return fmt.Errorf("platformk8s/providerconnect: client is nil")
	case config.Reader == nil:
		return fmt.Errorf("platformk8s/providerconnect: reader is nil")
	case config.Namespace == "":
		return fmt.Errorf("platformk8s/providerconnect: namespace is empty")
	case config.Credentials == nil:
		return fmt.Errorf("platformk8s/providerconnect: credential service is nil")
	case config.Providers == nil:
		return fmt.Errorf("platformk8s/providerconnect: provider service is nil")
	case config.ProviderReader == nil:
		return fmt.Errorf("platformk8s/providerconnect: provider reader is nil")
	case config.Surfaces == nil:
		return fmt.Errorf("platformk8s/providerconnect: provider surface service is nil")
	case config.VendorSupport == nil:
		return fmt.Errorf("platformk8s/providerconnect: vendor support service is nil")
	case config.CLISupport == nil:
		return fmt.Errorf("platformk8s/providerconnect: cli support service is nil")
	case config.PostConnect == nil:
		return fmt.Errorf("platformk8s/providerconnect: post-connect workflow runtime is nil")
	case config.OAuthSessions == nil:
		return fmt.Errorf("platformk8s/providerconnect: oauth session manager is nil")
	default:
		return nil
	}
}
