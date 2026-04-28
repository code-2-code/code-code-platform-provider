package providerconnect

import (
	"context"
	"log/slog"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

// CredentialAPIKeyCreate carries an API key credential create request.
type CredentialAPIKeyCreate struct {
	CredentialID string
	DisplayName  string
	VendorID     string
	APIKey       string
}

// CredentialSessionCreate carries a session credential create request.
type CredentialSessionCreate struct {
	CredentialID string
	DisplayName  string
	VendorID     string
	SchemaID     string
	RequiredKeys []string
	Values       map[string]string
}

type providerCredentialService interface {
	CreateAPIKey(ctx context.Context, request CredentialAPIKeyCreate) (string, error)
	CreateSession(ctx context.Context, request CredentialSessionCreate) (string, error)
	Delete(ctx context.Context, credentialID string) error
}

type providerSurfaceBindingService interface {
	providerSurfaceBindingReader
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
	credentials providerCredentialService
	providers   providerSurfaceBindingService
}

func newProviderConnectResources(credentials providerCredentialService, providers providerSurfaceBindingService) providerConnectResources {
	return providerConnectResources{
		credentials: credentials,
		providers:   providers,
	}
}

func (r providerConnectResources) APIKeyConnectRuntime(logger *slog.Logger) apiKeyConnectRuntime {
	return apiKeyConnectRuntime{
		CreateCredential: r.credentials.CreateAPIKey,
		DeleteCredential: r.credentials.Delete,
		CreateProvider:   r.providers.CreateProvider,
		Logger:           logger,
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
