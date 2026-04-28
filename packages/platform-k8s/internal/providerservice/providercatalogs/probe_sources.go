package providercatalogs

import (
	"context"
	"strings"

	"code-code.internal/go-contract/domainerror"
	"code-code.internal/platform-k8s/internal/cliruntimeservice/cliversions"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
	"github.com/jackc/pgx/v5/pgxpool"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ProbeSourceConfig holds the dependencies for constructing the catalog source registry.
type ProbeSourceConfig struct {
	StatePool *pgxpool.Pool
	Reader    ctrlclient.Reader
	Namespace string
}

// newCatalogSourceRegistryFromConfig builds a registry of catalog probe sources.
func newCatalogSourceRegistryFromConfig(
	ctx context.Context,
	config ProbeSourceConfig,
) (*catalogSourceRegistry, error) {
	registry := newCatalogSourceRegistry()
	if err := registerSurfaceSources(registry); err != nil {
		return nil, err
	}
	cliSupport, err := clisupport.NewManagementService()
	if err != nil {
		return nil, err
	}
	versionStore, err := cliversions.NewPostgresStore(config.StatePool)
	if err != nil {
		return nil, err
	}
	if err := registerCLISources(ctx, registry, registerCLISourcesConfig{
		support:      cliSupport,
		reader:       config.Reader,
		versionStore: versionStore,
		namespace:    config.Namespace,
	}); err != nil {
		return nil, err
	}
	return registry, nil
}

// catalogProbeRefFromProbeID converts a probe ID string to a catalogSourceRef.
func catalogProbeRefFromProbeID(probeID string) (catalogSourceRef, error) {
	probeID = strings.TrimSpace(probeID)
	if probeID == "" {
		return catalogSourceRef{}, domainerror.NewValidation("platformk8s/providercatalogs: model catalog probe id is empty")
	}
	return newCatalogSourceRef(probeID), nil
}
