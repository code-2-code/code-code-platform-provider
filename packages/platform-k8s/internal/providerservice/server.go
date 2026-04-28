package providerservice

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providercatalogs"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"code-code.internal/platform-k8s/internal/providerservice/providersurfacebindings"
	cliidentity "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/identity"
	"code-code.internal/platform-k8s/internal/supportservice/providersurfaces"
	"code-code.internal/platform-k8s/internal/supportservice/templates"
	vendoridentity "code-code.internal/platform-k8s/internal/supportservice/vendors/identity"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const actionStatusOK = "ok"

type Config struct {
	Client                             ctrlclient.Client
	Reader                             ctrlclient.Reader
	Namespace                          string
	AuthConn                           *grpc.ClientConn
	ModelConn                          *grpc.ClientConn
	StatePool                          *pgxpool.Pool
	ProviderConnectProviderHTTPBaseURL string
	ProviderHostTelemetryMaxTargets    int
	PostConnect                        providerconnect.PostConnectWorkflowRuntime
	Logger                             *slog.Logger
}

type Server struct {
	providerservicev1.UnimplementedProviderServiceServer

	surfaceMetadata         *providersurfaces.Service
	authClient              authv1.AuthServiceClient
	providers               providerManagementService
	providerSurfaceBindings *providersurfacebindings.Service
	providerConnect         *providerconnect.Service
	vendors                 *vendoridentity.VendorManagementService
	vendorSupport           *vendorsupport.ManagementService
	cliDefinitions          *cliidentity.CLIDefinitionManagementService
	templates               *templates.TemplateManagementService
	providerObservability   *providerobservability.Service
	catalogDiscovery        *providercatalogs.MaterializationSyncer
	providerHostTargetLimit int
}

type providerManagementService interface {
	List(context.Context) ([]*managementv1.ProviderView, error)
	Get(context.Context, string) (*managementv1.ProviderView, error)
	Update(context.Context, string, providers.UpdateProviderCommand) (*managementv1.ProviderView, error)
	UpdateAPIKeyAuthentication(context.Context, string, providers.UpdateAPIKeyAuthenticationCommand) (*managementv1.UpdateProviderAuthenticationResponse, error)
	UpdateObservabilityAuthentication(context.Context, string, providers.UpdateObservabilityAuthenticationCommand) (*managementv1.ProviderView, error)
	Delete(context.Context, string) error
}

func NewServer(config Config) (*Server, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("platformk8s/providerservice: client is nil")
	}
	if config.Reader == nil {
		config.Reader = config.Client
	}
	if strings.TrimSpace(config.Namespace) == "" {
		return nil, fmt.Errorf("platformk8s/providerservice: namespace is empty")
	}
	if config.AuthConn == nil {
		return nil, fmt.Errorf("platformk8s/providerservice: auth service connection is nil")
	}
	if config.StatePool == nil {
		return nil, fmt.Errorf("platformk8s/providerservice: state pool is nil")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.ProviderHostTelemetryMaxTargets <= 0 {
		config.ProviderHostTelemetryMaxTargets = 200
	}
	return assembleServer(config)
}
