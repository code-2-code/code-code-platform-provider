package providercatalogs

import (
	"context"
	"fmt"
	"strings"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"code-code.internal/platform-k8s/internal/cliruntimeservice/cliversions"
	clioauth "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/oauth"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CLISupportReader lists CLI support definitions.
type CLISupportReader interface {
	List(context.Context) ([]*supportv1.CLI, error)
}

type registerCLISourcesConfig struct {
	support      CLISupportReader
	reader       ctrlclient.Reader
	versionStore cliversions.Store
	namespace    string
}

func registerCLISources(ctx context.Context, registry *catalogSourceRegistry, config registerCLISourcesConfig) error {
	if config.support == nil {
		return fmt.Errorf("platformk8s/providercatalogs: cli support reader is nil")
	}
	clis, err := config.support.List(ctx)
	if err != nil {
		return err
	}
	for _, cli := range clis {
		cliID := strings.TrimSpace(cli.GetCliId())
		if cliID == "" {
			return fmt.Errorf("platformk8s/providercatalogs: cli support id is empty")
		}
		if hasDefaultCLICatalog(cli) {
			if err := registry.register(&cliCatalogSource{
				sourceRef: newCatalogSourceRef("cli." + cliID),
			}); err != nil {
				return err
			}
		}
		if _, operation, err := clioauth.ResolveOAuthModelCatalogDiscovery(cli); err != nil {
			return err
		} else if operation != nil {
			probeID := strings.TrimSpace(clisupport.OAuthModelCatalogProbeID(cli))
			if probeID == "" {
				continue
			}
			if err := registry.register(&cliCatalogSource{
				sourceRef: newCatalogSourceRef(probeID),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasDefaultCLICatalog(cli *supportv1.CLI) bool {
	if cli == nil || cli.GetOauth() == nil || cli.GetOauth().GetModelCatalog() == nil {
		return false
	}
	return cli.GetOauth().GetModelCatalog().GetDefaultCatalog() != nil
}

type cliCatalogSource struct {
	sourceRef catalogSourceRef
}

func (s *cliCatalogSource) ref() catalogSourceRef {
	return s.sourceRef
}
