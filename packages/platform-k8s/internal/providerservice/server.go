package providerservice

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providercatalogs"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"code-code.internal/platform-k8s/internal/providerservice/templates"
	"github.com/jackc/pgx/v5/pgxpool"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const actionStatusOK = "ok"

type Config struct {
	Client                          ctrlclient.Client
	Reader                          ctrlclient.Reader
	Namespace                       string
	StatePool                       *pgxpool.Pool
	ProviderHostTelemetryMaxTargets int
	Logger                          *slog.Logger
}

type Server struct {
	providerservicev1.UnimplementedProviderServiceServer

	surfaceMetadata         SurfaceRegistry
	providers               providerManagementService
	templates               *templates.TemplateManagementService
	providerObservability   *providerobservability.Service
	catalogDiscovery        *providercatalogs.MaterializationSyncer
	providerHostTargetLimit int
}

type providerManagementService interface {
	List(context.Context) ([]*managementv1.ProviderView, error)
	Get(context.Context, string) (*managementv1.ProviderView, error)
	Update(context.Context, string, providers.UpdateProviderCommand) (*managementv1.ProviderView, error)
	ApplyModelCatalog(context.Context, string, []*providerv1.ProviderModel) (*managementv1.ProviderView, error)
	ApplyProbeStatus(context.Context, string, providerv1.ProviderProbeKind, *providerv1.ProviderProbeRunState) (*managementv1.ProviderView, error)
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
