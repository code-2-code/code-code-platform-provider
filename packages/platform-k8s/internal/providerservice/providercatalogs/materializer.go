package providercatalogs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	modelv1 "code-code.internal/go-contract/model/v1"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/modelservice/modelidentity"
	"code-code.internal/platform-k8s/internal/platform/modelcatalogdiscovery"
	vendorsupport "code-code.internal/platform-k8s/internal/platform/vendors/support"
	"google.golang.org/protobuf/proto"
)

// ModelIDFilterInput describes one discovered provider model before it is
// accepted into the materialized catalog.
type ModelIDFilterInput struct {
	VendorID        string
	SurfaceID       string
	ProviderModelID string
}

// ModelIDFilter decides whether a provider model ID should be included in the
// materialized catalog. Return true to include, false to exclude.
type ModelIDFilter func(ModelIDFilterInput) bool

// ModelIDProbe discovers available model IDs for a provider surface.
type ModelIDProbe interface {
	ProbeModelIDs(ctx context.Context, request ProbeRequest) ([]string, error)
}

// ProbeRequest describes what to probe.
type ProbeRequest struct {
	ProbeID              string
	TargetID             string
	BaseURL              string
	Protocol             apiprotocolv1.Protocol
	SurfaceID            string
	ProviderCredentialID string
	Operation            *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation
}

// CatalogMaterializer refreshes provider surface catalogs by probing for model IDs.
type CatalogMaterializer struct {
	probe       ModelIDProbe
	modelFilter ModelIDFilter
	surfaces    SurfaceReader
	logger      *slog.Logger
}

type SurfaceReader interface {
	Get(context.Context, string) (*supportv1.Surface, error)
}

// NewCatalogMaterializer creates a materializer that probes for model IDs.
// An optional ModelIDFilter controls which discovered model IDs are included
// in the materialized catalog. If filter is nil, all model IDs are included.
func NewCatalogMaterializer(probe ModelIDProbe, logger *slog.Logger, filter ModelIDFilter, surfaces SurfaceReader) *CatalogMaterializer {
	if logger == nil {
		logger = slog.Default()
	}
	return &CatalogMaterializer{probe: probe, modelFilter: filter, surfaces: surfaces, logger: logger}
}

func (m *CatalogMaterializer) MaterializeProvider(ctx context.Context, provider *providerv1.Provider) (*providerv1.Provider, error) {
	if m == nil || m.probe == nil || provider == nil {
		return provider, nil
	}
	next := proto.Clone(provider).(*providerv1.Provider)
	if err := m.materializeProviderInternal(ctx, next); err != nil {
		return nil, fmt.Errorf("platformk8s/providercatalogs: materialize provider %q catalog: %w", next.GetProviderId(), err)
	}
	return next, nil
}

func (m *CatalogMaterializer) materializeProviderInternal(ctx context.Context, provider *providerv1.Provider) error {
	request, ok, err := m.providerProbeRequest(ctx, provider)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	modelIDs, err := m.probe.ProbeModelIDs(ctx, request)
	if err != nil {
		return err
	}
	surfaceID := strings.TrimSpace(provider.GetSurfaceId())
	modelIDs = m.filteredModelIDs(modelIDs, "", surfaceID)
	if modelsAlreadyCurrent(provider.GetModels(), modelIDs) {
		return nil
	}
	provider.Models = m.modelsFromModelIDs(provider.GetModels(), modelIDs, "")
	return nil
}

func (m *CatalogMaterializer) providerProbeRequest(ctx context.Context, provider *providerv1.Provider) (ProbeRequest, bool, error) {
	if provider == nil || m == nil || m.surfaces == nil {
		return ProbeRequest{}, false, nil
	}
	surfaceID := strings.TrimSpace(provider.GetSurfaceId())
	if surfaceID == "" {
		return ProbeRequest{}, false, nil
	}
	if request, ok := customAPIKeyProbeRequest(provider, surfaceID); ok {
		return request, true, nil
	}
	surface, err := m.surfaces.Get(ctx, surfaceID)
	if err != nil {
		return ProbeRequest{}, false, err
	}
	probeID := strings.TrimSpace(vendorsupport.SurfaceModelCatalogProbeID(surface))
	if probeID == "" {
		return ProbeRequest{}, false, nil
	}
	endpoint, ok := vendorsupport.SurfaceDefaultAPIEndpoint(surface)
	if !ok {
		return ProbeRequest{}, false, nil
	}
	request := ProbeRequest{
		ProbeID:              probeID,
		TargetID:             strings.TrimSpace(provider.GetProviderId()),
		SurfaceID:            surfaceID,
		ProviderCredentialID: strings.TrimSpace(provider.GetProviderCredentialRef().GetProviderCredentialId()),
		BaseURL:              strings.TrimSpace(endpoint.GetBaseUrl()),
		Protocol:             endpoint.GetProtocol(),
	}
	if custom := provider.GetCustomApiKeySurface(); custom != nil {
		if baseURL := strings.TrimSpace(custom.GetBaseUrl()); baseURL != "" {
			request.BaseURL = baseURL
		}
		if protocol := custom.GetProtocol(); protocol != apiprotocolv1.Protocol_PROTOCOL_UNSPECIFIED {
			request.Protocol = protocol
		}
	}
	request.Operation = modelCatalogProbeOperation(probeID, request.Protocol, request.BaseURL)
	return request, true, nil
}

func customAPIKeyProbeRequest(provider *providerv1.Provider, surfaceID string) (ProbeRequest, bool) {
	if strings.TrimSpace(surfaceID) != "custom.api" {
		return ProbeRequest{}, false
	}
	custom := provider.GetCustomApiKeySurface()
	if custom == nil {
		return ProbeRequest{}, false
	}
	probeID := customAPIKeyModelCatalogProbeID(custom.GetProtocol())
	if probeID == "" {
		return ProbeRequest{}, false
	}
	return ProbeRequest{
		ProbeID:              probeID,
		TargetID:             strings.TrimSpace(provider.GetProviderId()),
		SurfaceID:            strings.TrimSpace(surfaceID),
		ProviderCredentialID: strings.TrimSpace(provider.GetProviderCredentialRef().GetProviderCredentialId()),
		BaseURL:              strings.TrimSpace(custom.GetBaseUrl()),
		Protocol:             custom.GetProtocol(),
	}, true
}

func customAPIKeyModelCatalogProbeID(protocol apiprotocolv1.Protocol) string {
	switch protocol {
	case apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, apiprotocolv1.Protocol_PROTOCOL_OPENAI_RESPONSES:
		return "surface.openai-compatible"
	default:
		return ""
	}
}

func modelCatalogProbeOperation(probeID string, protocol apiprotocolv1.Protocol, baseURL string) *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation {
	if strings.TrimSpace(probeID) == "surface.cloudflare-workers-ai" {
		baseURL = strings.TrimSpace(strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(baseURL), "/"), "/v1"))
		return &modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation{
			BaseUrl:      baseURL,
			Path:         "models/search",
			ResponseKind: modelcatalogdiscoveryv1.ModelCatalogDiscoveryResponseKind_MODEL_CATALOG_DISCOVERY_RESPONSE_KIND_OPENAI_MODELS,
			Security: []*modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityRequirement{{
				Schemes: []modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme{
					modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_API_KEY,
				},
			}},
		}
	}
	return modelcatalogdiscovery.DefaultAPIKeyDiscoveryOperation(protocol)
}

func (m *CatalogMaterializer) filteredModelIDs(modelIDs []string, vendorID string, surfaceID string) []string {
	if m.modelFilter == nil {
		return modelIDs
	}
	out := make([]string, 0, len(modelIDs))
	for _, rawModelID := range modelIDs {
		providerModelID := strings.TrimSpace(rawModelID)
		if providerModelID == "" {
			continue
		}
		if !m.modelFilter(ModelIDFilterInput{
			VendorID:        strings.TrimSpace(vendorID),
			SurfaceID:       strings.TrimSpace(surfaceID),
			ProviderModelID: providerModelID,
		}) {
			continue
		}
		out = append(out, providerModelID)
	}
	return out
}

// modelsFromModelIDs builds provider-callable models from discovered model IDs.
// It does inline best-effort binding: if a model ID can be resolved to a canonical
// ModelRef via identity normalization, it is bound immediately.
func (m *CatalogMaterializer) modelsFromModelIDs(current []*providerv1.ProviderModel, modelIDs []string, vendorID string) []*providerv1.ProviderModel {
	existingRefs := existingModelRefs(current)
	models := make([]*providerv1.ProviderModel, 0, len(modelIDs))
	for _, rawModelID := range modelIDs {
		providerModelID := strings.TrimSpace(rawModelID)
		if providerModelID == "" {
			continue
		}
		modelRef := existingRefs[providerModelID]
		if modelRef == nil {
			modelRef = resolveModelRef(vendorID, providerModelID)
		}
		models = append(models, &providerv1.ProviderModel{
			ProviderModelId: providerModelID,
			ModelRef:        modelRef,
		})
	}
	return models
}

// resolveModelRef attempts best-effort identity resolution for a provider model ID.
// Returns nil if the model cannot be mapped to a canonical reference.
func resolveModelRef(vendorID string, providerModelID string) *modelv1.ModelRef {
	if strings.TrimSpace(vendorID) == "" {
		return nil
	}
	slug := modelidentity.NormalizeExternalModelSlug(providerModelID)
	if slug == "" || modelidentity.HasChannelToken(slug) {
		return nil
	}
	candidates := modelidentity.ExternalModelCandidates(slug)
	if len(candidates) == 0 {
		return nil
	}
	// Use the first non-raw candidate (stripped of snapshot suffix) as the canonical model ID.
	canonicalModelID := candidates[0]
	return &modelv1.ModelRef{
		VendorId: vendorID,
		ModelId:  canonicalModelID,
	}
}

func modelsAlreadyCurrent(current []*providerv1.ProviderModel, modelIDs []string) bool {
	if len(current) != len(modelIDs) {
		return false
	}
	for index, modelID := range modelIDs {
		if strings.TrimSpace(current[index].GetProviderModelId()) != strings.TrimSpace(modelID) {
			return false
		}
	}
	return true
}

func existingModelRefs(models []*providerv1.ProviderModel) map[string]*modelv1.ModelRef {
	out := map[string]*modelv1.ModelRef{}
	for _, item := range models {
		modelID := strings.TrimSpace(item.GetProviderModelId())
		if modelID == "" || item.GetModelRef() == nil {
			continue
		}
		out[modelID] = proto.Clone(item.GetModelRef()).(*modelv1.ModelRef)
	}
	return out
}
