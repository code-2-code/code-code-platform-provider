package providerorchestration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"

	providerv1 "code-code.internal/go-contract/provider/v1"
	clisupport "code-code.internal/platform-k8s/internal/platform/clidefinitions/support"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
	vendorsupport "code-code.internal/platform-k8s/internal/platform/vendors/support"
	"code-code.internal/platform-k8s/internal/providerconnect"
	"code-code.internal/platform-k8s/internal/providerpostconnect"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/proto"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ConnectRuntimeConfig struct {
	Client                  ctrlclient.Client
	Reader                  ctrlclient.Reader
	Namespace               string
	StatePool               *pgxpool.Pool
	Auth                    authv1.AuthServiceClient
	OAuth                   oauthv1.OAuthSessionServiceClient
	TemporalClient          client.Client
	PostConnectTaskQueue    string
	ProviderHTTPBaseURL     string
	ProviderHTTPActionToken string
	Logger                  *slog.Logger
}

func NewProviderConnectRuntime(config ConnectRuntimeConfig) (*providerconnect.Service, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: kubernetes client is nil")
	}
	if config.Reader == nil {
		config.Reader = config.Client
	}
	if strings.TrimSpace(config.Namespace) == "" {
		return nil, fmt.Errorf("platformk8s/providerorchestration: namespace is empty")
	}
	if config.StatePool == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: state pool is nil")
	}
	if config.Auth == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: auth client is nil")
	}
	if config.OAuth == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: oauth client is nil")
	}
	if config.TemporalClient == nil {
		return nil, fmt.Errorf("platformk8s/providerorchestration: temporal client is nil")
	}
	surfaceMetadata, err := providersurfaces.NewService()
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
	providerAccounts, err := providers.NewService(providers.Config{
		StatePool: config.StatePool,
		Surfaces:  surfaceMetadata,
	})
	if err != nil {
		return nil, err
	}
	postConnect, err := providerpostconnect.NewTemporalWorkflowRuntime(providerpostconnect.TemporalWorkflowRuntimeConfig{
		Client:                  config.TemporalClient,
		TaskQueue:               config.PostConnectTaskQueue,
		PlatformNamespace:       config.Namespace,
		ProviderHTTPBaseURL:     config.ProviderHTTPBaseURL,
		ProviderHTTPActionToken: config.ProviderHTTPActionToken,
	})
	if err != nil {
		return nil, err
	}
	return providerconnect.NewService(providerconnect.Config{
		Client:         config.Client,
		Reader:         config.Reader,
		Namespace:      config.Namespace,
		Providers:      orchestrationProviderAdapter{source: providerAccounts},
		ProviderReader: orchestrationProviderAdapter{source: providerAccounts},
		Surfaces:       surfaceMetadata,
		VendorSupport:  vendorSupport,
		CLISupport:     cliSupport,
		PostConnect:    postConnect,
		OAuthSessions:  orchestrationOAuthSessionService{client: config.OAuth},
		Logger:         config.Logger,
	})
}

type orchestrationOAuthSessionService struct {
	client oauthv1.OAuthSessionServiceClient
}

func (s orchestrationOAuthSessionService) StartSession(ctx context.Context, request *credentialv1.OAuthAuthorizationSessionSpec) (*credentialv1.OAuthAuthorizationSessionState, error) {
	response, err := s.client.StartOAuthAuthorizationSession(ctx, &oauthv1.StartOAuthAuthorizationSessionRequest{
		CliId:              strings.TrimSpace(request.GetCliId()),
		Flow:               request.GetFlow(),
		TargetCredentialId: strings.TrimSpace(request.GetTargetCredentialId()),
		TargetDisplayName:  strings.TrimSpace(request.GetTargetDisplayName()),
	})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (s orchestrationOAuthSessionService) GetSession(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	response, err := s.client.GetOAuthAuthorizationSession(ctx, &oauthv1.GetOAuthAuthorizationSessionRequest{
		SessionId: strings.TrimSpace(sessionID),
	})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

func (s orchestrationOAuthSessionService) CancelSession(ctx context.Context, sessionID string) (*credentialv1.OAuthAuthorizationSessionState, error) {
	response, err := s.client.CancelOAuthAuthorizationSession(ctx, &oauthv1.CancelOAuthAuthorizationSessionRequest{
		SessionId: strings.TrimSpace(sessionID),
	})
	if err != nil {
		return nil, err
	}
	return response.GetSession(), nil
}

type orchestrationProviderAdapter struct {
	source *providers.Service
}

func (a orchestrationProviderAdapter) Get(ctx context.Context, providerID string) (*providerconnect.ProviderView, error) {
	view, err := a.source.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return providerConnectProviderFromManagement(view), nil
}

func (a orchestrationProviderAdapter) List(ctx context.Context) ([]*providerconnect.ProviderView, error) {
	items, err := a.source.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*providerconnect.ProviderView, 0, len(items))
	for _, item := range items {
		if next := providerConnectProviderFromManagement(item); next != nil {
			out = append(out, next)
		}
	}
	return out, nil
}

func (a orchestrationProviderAdapter) CreateProvider(ctx context.Context, provider *providerv1.Provider) (*providerconnect.ProviderView, error) {
	view, err := a.source.CreateProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	return providerConnectProviderFromManagement(view), nil
}

func providerConnectProviderFromManagement(view *managementv1.ProviderView) *providerconnect.ProviderView {
	if view == nil {
		return nil
	}
	out := &providerconnect.ProviderView{
		ProviderID:           strings.TrimSpace(view.GetProviderId()),
		DisplayName:          strings.TrimSpace(view.GetDisplayName()),
		SurfaceID:            strings.TrimSpace(view.GetSurfaceId()),
		ProviderCredentialID: strings.TrimSpace(view.GetProviderCredentialId()),
		ProductInfoID:        strings.TrimSpace(view.GetProductInfoId()),
	}
	if runtime := view.GetRuntime(); runtime != nil {
		out.Runtime = proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
	}
	return out
}
