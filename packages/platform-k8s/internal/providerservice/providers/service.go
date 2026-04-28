package providers

import (
	"context"
	"fmt"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CredentialAPIKeyUpdate carries an API key credential update request.
type CredentialAPIKeyUpdate struct {
	CredentialID string
	DisplayName  string
	Purpose      credentialv1.CredentialPurpose
	VendorID     string
	APIKey       string
}

// CredentialSessionUpdate carries a session credential update request.
type CredentialSessionUpdate struct {
	CredentialID string
	DisplayName  string
	Purpose      credentialv1.CredentialPurpose
	VendorID     string
	SchemaID     string
	RequiredKeys []string
	Values       map[string]string
	MergeValues  bool
}

type credentialService interface {
	UpdateAPIKey(ctx context.Context, request CredentialAPIKeyUpdate) (*managementv1.CredentialView, error)
	UpdateSession(ctx context.Context, request CredentialSessionUpdate) (*managementv1.CredentialView, error)
	Rename(ctx context.Context, credentialID, displayName string) error
	Delete(ctx context.Context, credentialID string) error
	Exists(ctx context.Context, credentialID string) (bool, error)
	CredentialSubjectSummary(ctx context.Context, credentialID string) ([]*managementv1.CredentialSubjectSummaryFieldView, error)
}

type vendorReferenceService interface {
	List(ctx context.Context) ([]*managementv1.VendorView, error)
}

type cliDefinitionReferenceService interface {
	List(ctx context.Context) ([]*managementv1.CLIDefinitionView, error)
}

type cliSupportReferenceService interface {
	Get(ctx context.Context, cliID string) (*supportv1.CLI, error)
}

type Service struct {
	repository  Store
	credentials credentialService
	vendors     vendorReferenceService
	cliDefs     cliDefinitionReferenceService
	cliSupport  cliSupportReferenceService
}

type Config struct {
	StatePool   *pgxpool.Pool
	Credentials credentialService
	Vendors     vendorReferenceService
	CLIDefs     cliDefinitionReferenceService
	CLISupport  cliSupportReferenceService
}

func NewService(config Config) (*Service, error) {
	switch {
	case config.StatePool == nil:
		return nil, fmt.Errorf("platformk8s/providers: state pool is nil")
	case config.Credentials == nil:
		return nil, fmt.Errorf("platformk8s/providers: credential service is nil")
	}
	repository, err := NewProviderRepository(config.StatePool)
	if err != nil {
		return nil, err
	}
	return &Service{
		repository:  repository,
		credentials: config.Credentials,
		vendors:     config.Vendors,
		cliDefs:     config.CLIDefs,
		cliSupport:  config.CLISupport,
	}, nil
}
