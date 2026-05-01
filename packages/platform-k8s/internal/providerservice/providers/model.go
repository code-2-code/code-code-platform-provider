package providers

import (
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
	"google.golang.org/protobuf/proto"
)

type ProviderProjection struct {
	value *managementv1.ProviderView
}

func providerProjectionFromProvider(provider *providerv1.Provider, surface *supportv1.Surface) *ProviderProjection {
	view := &managementv1.ProviderView{
		ProviderId:           strings.TrimSpace(provider.GetProviderId()),
		DisplayName:          strings.TrimSpace(provider.GetDisplayName()),
		ProviderCredentialId: strings.TrimSpace(provider.GetProviderCredentialRef().GetProviderCredentialId()),
		Models:               cloneProviderModels(provider.GetModels()),
		ProbeStatus:          cloneProviderProbeStatus(provider.GetProbeStatus()),
		SurfaceId:            strings.TrimSpace(provider.GetSurfaceId()),
		Endpoints:            providerEndpoints(provider, surface),
		ProductInfoId:        strings.TrimSpace(surface.GetProductInfoId()),
		Status:               statusFromProvider(provider),
	}
	return &ProviderProjection{value: view}
}

func providerEndpoints(provider *providerv1.Provider, surface *supportv1.Surface) []*providerv1.ProviderEndpoint {
	if custom := provider.GetCustomApiKeySurface(); custom != nil {
		endpoint := &providerv1.ProviderEndpoint{
			Type: providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_API,
			Shape: &providerv1.ProviderEndpoint_Api{Api: &providerv1.ProviderApiEndpoint{
				BaseUrl:  strings.TrimSpace(custom.GetBaseUrl()),
				Protocol: custom.GetProtocol(),
			}},
		}
		if err := providerv1.ValidateProviderEndpoint(endpoint); err != nil {
			return nil
		}
		return []*providerv1.ProviderEndpoint{endpoint}
	}
	return cloneProviderEndpoints(providersurfaces.MaterializeEndpoints(surface))
}

func statusFromProvider(provider *providerv1.Provider) *managementv1.ProviderStatus {
	if provider == nil {
		return &managementv1.ProviderStatus{
			Phase:  providerservicev1.ProviderPhase_PROVIDER_PHASE_INVALID_CONFIG,
			Reason: "provider is nil",
		}
	}
	if err := providerv1.ValidateProvider(provider); err != nil {
		return &managementv1.ProviderStatus{
			Phase:  providerservicev1.ProviderPhase_PROVIDER_PHASE_INVALID_CONFIG,
			Reason: err.Error(),
		}
	}
	return &managementv1.ProviderStatus{
		Phase: providerservicev1.ProviderPhase_PROVIDER_PHASE_READY,
	}
}

func (p *ProviderProjection) Proto() *managementv1.ProviderView {
	if p == nil || p.value == nil {
		return &managementv1.ProviderView{}
	}
	return proto.Clone(p.value).(*managementv1.ProviderView)
}

func (p *ProviderProjection) ID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetProviderId())
}

func (p *ProviderProjection) DisplayName() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetDisplayName())
}

func (p *ProviderProjection) CredentialID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetProviderCredentialId())
}

func (p *ProviderProjection) SurfaceID() string {
	if p == nil || p.value == nil {
		return ""
	}
	return strings.TrimSpace(p.value.GetSurfaceId())
}

func (p *ProviderProjection) AuthKind() providerv1.ProviderEndpointType {
	if p == nil || p.value == nil || len(p.value.GetEndpoints()) == 0 {
		return providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_UNSPECIFIED
	}
	return p.value.GetEndpoints()[0].GetType()
}

func (p *ProviderProjection) CLIID() string {
	if p == nil || p.value == nil {
		return ""
	}
	for _, endpoint := range p.value.GetEndpoints() {
		if cliID := strings.TrimSpace(endpoint.GetCli().GetCliId()); cliID != "" {
			return cliID
		}
	}
	return ""
}

func cloneProviderModels(models []*providerv1.ProviderModel) []*providerv1.ProviderModel {
	if models == nil {
		return nil
	}
	out := make([]*providerv1.ProviderModel, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		out = append(out, proto.Clone(model).(*providerv1.ProviderModel))
	}
	return out
}

func cloneProviderEndpoints(endpoints []*providerv1.ProviderEndpoint) []*providerv1.ProviderEndpoint {
	if endpoints == nil {
		return nil
	}
	out := make([]*providerv1.ProviderEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint == nil {
			continue
		}
		out = append(out, proto.Clone(endpoint).(*providerv1.ProviderEndpoint))
	}
	return out
}

func cloneProviderProbeStatus(status *providerv1.ProviderProbeStatus) *providerv1.ProviderProbeStatus {
	if status == nil {
		return nil
	}
	return proto.Clone(status).(*providerv1.ProviderProbeStatus)
}
