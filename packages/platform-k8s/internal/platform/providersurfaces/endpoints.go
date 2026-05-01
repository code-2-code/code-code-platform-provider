package providersurfaces

import (
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

// Endpoints derives provider call endpoints from one provider surface definition.
func Endpoints(surface *supportv1.Surface) []*providerv1.ProviderEndpoint {
	if surface == nil {
		return nil
	}
	if cliID := strings.TrimSpace(surface.GetCli().GetCliId()); cliID != "" {
		return []*providerv1.ProviderEndpoint{{
			Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_CLI,
			Shape: &providerv1.ProviderEndpoint_Cli{Cli: &providerv1.ProviderCliEndpoint{
				CliId: cliID,
			}},
		}}
	}
	out := []*providerv1.ProviderEndpoint{}
	for _, endpoint := range surface.GetApi().GetApiEndpoints() {
		if endpoint == nil || endpoint.GetProtocol() == apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED || strings.TrimSpace(endpoint.GetBaseUrl()) == "" {
			continue
		}
		out = append(out, &providerv1.ProviderEndpoint{
			Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_API,
			Shape: &providerv1.ProviderEndpoint_Api{Api: &providerv1.ProviderApiEndpoint{
				BaseUrl:  strings.TrimSpace(endpoint.GetBaseUrl()),
				Protocol: endpoint.GetProtocol(),
			}},
		})
	}
	return out
}

// MaterializeEndpoints returns cloned provider call endpoints for projection.
func MaterializeEndpoints(surface *supportv1.Surface) []*providerv1.ProviderEndpoint {
	endpoints := Endpoints(surface)
	out := make([]*providerv1.ProviderEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint == nil {
			continue
		}
		out = append(out, proto.Clone(endpoint).(*providerv1.ProviderEndpoint))
	}
	return out
}
