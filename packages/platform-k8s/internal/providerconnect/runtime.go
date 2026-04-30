package providerconnect

import (
	"context"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

type providerCreationService interface {
	CreateProvider(ctx context.Context, provider *providerv1.Provider) (*ProviderView, error)
}

type vendorSupportReader interface {
	GetForConnect(ctx context.Context, vendorID string) (*supportv1.Vendor, error)
}

type cliSupportReader interface {
	Get(ctx context.Context, cliID string) (*supportv1.CLI, error)
}

type providerConnectSessionStore interface {
	create(ctx context.Context, record *sessionRecord) error
	get(ctx context.Context, sessionID string) (*sessionRecord, error)
	put(ctx context.Context, record *sessionRecord) error
}

type providerConnectResources struct {
	providers providerCreationService
}

func newProviderConnectResources(providers providerCreationService) providerConnectResources {
	return providerConnectResources{
		providers: providers,
	}
}

func (r providerConnectResources) APIKeyConnectRuntime() apiKeyConnectRuntime {
	return apiKeyConnectRuntime{
		CreateProvider: r.providers.CreateProvider,
	}
}

type providerConnectSupport struct {
	vendors vendorSupportReader
	clis    cliSupportReader
}

func newProviderConnectSupport(vendors vendorSupportReader, clis cliSupportReader) providerConnectSupport {
	return providerConnectSupport{
		vendors: vendors,
		clis:    clis,
	}
}

type providerConnectSessions struct {
	oauth oauthSessionService
	store providerConnectSessionStore
}

func newProviderConnectSessions(oauth oauthSessionService, store providerConnectSessionStore) providerConnectSessions {
	return providerConnectSessions{
		oauth: oauth,
		store: store,
	}
}
