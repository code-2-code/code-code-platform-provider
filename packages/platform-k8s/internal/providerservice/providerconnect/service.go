package providerconnect

import (
	"context"
	"log/slog"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type providerReader interface {
	Get(ctx context.Context, providerID string) (*ProviderView, error)
}

type surfaceMetadataReader interface {
	Get(ctx context.Context, surfaceID string) (*providerv1.ProviderSurface, error)
}

type providerSurfaceBindingReader interface {
	ListProviderSurfaceBindings(ctx context.Context) ([]*ProviderSurfaceBindingView, error)
}

type oauthSessionService interface {
	StartSession(ctx context.Context, request *credentialv1.OAuthAuthorizationSessionSpec) (*credentialv1.OAuthAuthorizationSessionState, error)
	GetSession(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error)
	CancelSession(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error)
}

type Service struct {
	runtime providerConnectRuntime
}

type Config struct {
	Client         ctrlclient.Client
	Reader         ctrlclient.Reader
	Namespace      string
	Credentials    providerCredentialService
	Providers      providerSurfaceBindingService
	ProviderReader providerReader
	Surfaces       surfaceMetadataReader
	VendorSupport  *vendorsupport.ManagementService
	CLISupport     *clisupport.ManagementService
	PostConnect    PostConnectWorkflowRuntime
	OAuthSessions  oauthSessionService
	Logger         *slog.Logger
}

func NewService(config Config) (*Service, error) {
	config.Namespace = strings.TrimSpace(config.Namespace)
	if err := validateProviderConnectConfig(config); err != nil {
		return nil, err
	}
	runtime, err := newProviderConnectRuntime(config)
	if err != nil {
		return nil, err
	}
	return &Service{
		runtime: runtime,
	}, nil
}

func (s *Service) Connect(ctx context.Context, command *ConnectCommand) (*ConnectResult, error) {
	if command == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: connect command is nil")
	}
	switch command.AddMethod() {
	case AddMethodAPIKey:
		return s.runtime.connectWithAPIKey(ctx, command)
	case AddMethodCLIOAuth:
		return s.runtime.connectWithCLIOAuth(ctx, command)
	default:
		return nil, domainerror.NewValidation("platformk8s/providerconnect: add_method is required")
	}
}

func (s *Service) GetSession(ctx context.Context, sessionID string) (*SessionView, error) {
	return s.runtime.sessionQueryRuntime().Get(ctx, sessionID)
}

func (s *Service) Reauthorize(ctx context.Context, provider *ProviderView) (*SessionView, error) {
	return s.runtime.Reauthorize(ctx, provider)
}
