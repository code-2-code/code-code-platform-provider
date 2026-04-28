package providercatalogs

import (
	"context"
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/platform-k8s/internal/modelservice/modelcatalogdiscovery"
)

// materializerProbe adapts CatalogProbeExecutor to the ModelIDProbe interface
// used by the materializer. It translates the simplified ProbeRequest into a
// full CatalogProbeRequest.
type materializerProbe struct {
	executor *CatalogProbeExecutor
}

// NewMaterializerProbe creates a ModelIDProbe backed by a CatalogProbeExecutor.
func NewMaterializerProbe(executor *CatalogProbeExecutor) ModelIDProbe {
	return &materializerProbe{executor: executor}
}

func (p *materializerProbe) ProbeModelIDs(ctx context.Context, request ProbeRequest) ([]string, error) {
	protocol := parseProtocol(request.Protocol)
	internal := CatalogProbeRequest{
		ProbeID:                  request.ProbeID,
		Protocol:                 protocol,
		BaseURL:                  strings.TrimSpace(request.BaseURL),
		ProviderSurfaceBindingID: strings.TrimSpace(request.ProviderSurfaceBindingID),
		Operation:                modelcatalogdiscovery.DefaultAPIKeyDiscoveryOperation(protocol),
		ConcurrencyKey:           strings.TrimSpace(request.TargetID),
	}
	return p.executor.ProbeModelIDs(ctx, internal)
}

func parseProtocol(value string) apiprotocolv1.Protocol {
	value = strings.TrimSpace(value)
	if value == "" {
		return apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED
	}
	if parsed, ok := apiprotocolv1.Protocol_value[value]; ok {
		return apiprotocolv1.Protocol(parsed)
	}
	return apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED
}
