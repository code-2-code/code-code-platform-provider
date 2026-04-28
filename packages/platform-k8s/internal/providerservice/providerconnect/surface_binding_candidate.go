package providerconnect

import (
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/go-contract/domainerror"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
	"google.golang.org/protobuf/proto"
)

type connectSurfaceBindingCandidate struct {
	surfaceID string
	runtime   *providerv1.ProviderSurfaceRuntime
}

func newCustomAPIKeyCandidate(
	displayName string,
	material *APIKeyConnectInput,
	catalogs *surfaceCatalogSet,
) (*connectSurfaceBindingCandidate, error) {
	if material == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: api key material is required")
	}
	surfaceID := apiSurfaceIDForProtocol(material.Protocol)
	catalog := catalogs.Override(surfaceID, nil)
	return newConnectSurfaceBindingCandidate(surfaceID, &providerv1.ProviderSurfaceRuntime{
		DisplayName:         strings.TrimSpace(displayName),
		Origin:              providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL,
		Catalog:             catalog,
		ModelCatalogProbeId: surfaceModelCatalogProbeID(surfaceID),
		Access: &providerv1.ProviderSurfaceRuntime_Api{
			Api: &providerv1.ProviderAPISurfaceRuntime{
				Protocol: material.Protocol,
				BaseUrl:  strings.TrimSpace(material.BaseURL),
			},
		},
	}, "platformk8s/providerconnect: invalid custom provider surface binding")
}

func newVendorAPIKeyCandidates(
	vendor *supportv1.Vendor,
	catalogs *surfaceCatalogSet,
) ([]*connectSurfaceBindingCandidate, error) {
	if vendor == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: vendor support is nil")
	}
	out := []*connectSurfaceBindingCandidate{}
	for _, binding := range vendor.GetProviderBindings() {
		bindingSurfaceID := strings.TrimSpace(binding.GetProviderBinding().GetSurfaceId())
		for _, template := range binding.GetSurfaceTemplates() {
			if template == nil {
				continue
			}
			surfaceID := strings.TrimSpace(template.GetSurfaceId())
			if surfaceID == "" {
				surfaceID = bindingSurfaceID
			}
			runtime := cloneProviderSurfaceRuntime(template.GetRuntime())
			if runtime == nil {
				continue
			}
			runtime.Origin = providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED
			runtime.Catalog = catalogs.Override(surfaceID, template.GetBootstrapCatalog())
			candidate, err := newConnectSurfaceBindingCandidate(surfaceID, runtime, "platformk8s/providerconnect: invalid provider surface binding")
			if err != nil {
				return nil, err
			}
			out = append(out, candidate)
		}
	}
	if len(out) == 0 {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: vendor support does not expose any provider surface bindings")
	}
	if err := catalogs.ValidateAllMatched(); err != nil {
		return nil, err
	}
	return out, nil
}

func newCLIOAuthCandidate(displayName, cliID, surfaceID string, cli *supportv1.CLI) (*connectSurfaceBindingCandidate, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		surfaceID = clisupport.OAuthProviderSurfaceID(cli)
	}
	return newConnectSurfaceBindingCandidate(surfaceID, &providerv1.ProviderSurfaceRuntime{
		DisplayName:         strings.TrimSpace(displayName),
		Origin:              providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		ModelCatalogProbeId: clisupport.OAuthModelCatalogProbeID(cli),
		QuotaProbeId:        clisupport.OAuthQuotaProbeID(cli),
		EgressRulesetId:     clisupport.OAuthEgressPolicyID(cli),
		Access: &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: strings.TrimSpace(cliID)},
		},
	}, "platformk8s/providerconnect: invalid cli provider surface binding")
}

func newConnectSurfaceBindingCandidate(surfaceID string, runtime *providerv1.ProviderSurfaceRuntime, message string) (*connectSurfaceBindingCandidate, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return nil, domainerror.NewValidation("%s: surface_id is required", message)
	}
	if err := providerv1.ValidateProviderSurfaceRuntime(runtime); err != nil {
		return nil, domainerror.NewValidation("%s: %v", message, err)
	}
	return &connectSurfaceBindingCandidate{
		surfaceID: surfaceID,
		runtime:   cloneProviderSurfaceRuntime(runtime),
	}, nil
}

func apiSurfaceIDForProtocol(protocol apiprotocolv1.Protocol) string {
	switch protocol {
	case apiprotocolv1.Protocol_PROTOCOL_GEMINI:
		return "gemini"
	case apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC:
		return "anthropic"
	case apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, apiprotocolv1.Protocol_PROTOCOL_OPENAI_RESPONSES:
		return "openai-compatible"
	default:
		return ""
	}
}

func surfaceModelCatalogProbeID(surfaceID string) string {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return ""
	}
	return "surface." + surfaceID
}

func (c *connectSurfaceBindingCandidate) SurfaceID() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.surfaceID)
}

func (c *connectSurfaceBindingCandidate) Runtime() *providerv1.ProviderSurfaceRuntime {
	if c == nil {
		return nil
	}
	return cloneProviderSurfaceRuntime(c.runtime)
}

func cloneProviderSurfaceRuntime(runtime *providerv1.ProviderSurfaceRuntime) *providerv1.ProviderSurfaceRuntime {
	if runtime == nil {
		return nil
	}
	return proto.Clone(runtime).(*providerv1.ProviderSurfaceRuntime)
}
