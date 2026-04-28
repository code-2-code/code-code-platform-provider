package providercatalogs

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	modelv1 "code-code.internal/go-contract/model/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/modelservice/modelidentity"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ModelIDFilter decides whether a provider model ID should be included in
// the materialized catalog. Return true to include, false to exclude.
type ModelIDFilter func(providerModelID string) bool

// ModelIDProbe discovers available model IDs for a provider surface.
type ModelIDProbe interface {
	ProbeModelIDs(ctx context.Context, request ProbeRequest) ([]string, error)
}

// ProbeRequest describes what to probe.
type ProbeRequest struct {
	ProbeID                  string
	TargetID                 string
	BaseURL                  string
	Protocol                 string
	ProviderSurfaceBindingID string
}

// CatalogMaterializer refreshes provider surface catalogs by probing for model IDs.
type CatalogMaterializer struct {
	probe       ModelIDProbe
	modelFilter ModelIDFilter
	logger      *slog.Logger
}

// NewCatalogMaterializer creates a materializer that probes for model IDs.
// An optional ModelIDFilter controls which discovered model IDs are included
// in the materialized catalog. If filter is nil, all model IDs are included.
func NewCatalogMaterializer(probe ModelIDProbe, logger *slog.Logger, filter ModelIDFilter) *CatalogMaterializer {
	if logger == nil {
		logger = slog.Default()
	}
	return &CatalogMaterializer{probe: probe, modelFilter: filter, logger: logger}
}

// ExcludeByPattern returns a ModelIDFilter that excludes model IDs matching
// the given regular expression pattern. This is the common case for filtering
// out non-text-generation models (image, audio, embedding, etc.).
func ExcludeByPattern(pattern *regexp.Regexp) ModelIDFilter {
	if pattern == nil {
		return nil
	}
	return func(providerModelID string) bool {
		return !pattern.MatchString(providerModelID)
	}
}

// MaterializeProvider refreshes catalog entries for all surfaces of a provider.
func (m *CatalogMaterializer) MaterializeProvider(ctx context.Context, provider *providerv1.Provider) (*providerv1.Provider, error) {
	if m == nil || m.probe == nil || provider == nil {
		return provider, nil
	}
	next := proto.Clone(provider).(*providerv1.Provider)
	for _, surface := range next.GetSurfaces() {
		if err := m.materializeSurface(ctx, surface); err != nil {
			return nil, fmt.Errorf("platformk8s/providercatalogs: materialize surface %q catalog: %w", surface.GetSurfaceId(), err)
		}
	}
	return next, nil
}

func (m *CatalogMaterializer) materializeSurface(ctx context.Context, surface *providerv1.ProviderSurfaceBinding) error {
	request, ok := surfaceProbeRequest(surface)
	if !ok {
		return nil
	}
	modelIDs, err := m.probe.ProbeModelIDs(ctx, request)
	if err != nil {
		return err
	}
	current := surface.GetRuntime().GetCatalog()
	if catalogAlreadyCurrent(current, modelIDs) {
		return nil
	}
	vendorID := catalogOwnerVendorID(surface)
	catalog := m.catalogFromModelIDs(current, modelIDs, vendorID)
	runtime := proto.Clone(surface.GetRuntime()).(*providerv1.ProviderSurfaceRuntime)
	runtime.Catalog = catalog
	surface.Runtime = runtime
	return nil
}

func surfaceProbeRequest(surface *providerv1.ProviderSurfaceBinding) (ProbeRequest, bool) {
	if surface == nil || surface.GetRuntime() == nil {
		return ProbeRequest{}, false
	}
	runtime := surface.GetRuntime()
	probeID := strings.TrimSpace(runtime.GetModelCatalogProbeId())
	if probeID == "" {
		return ProbeRequest{}, false
	}
	source := surface.GetSourceRef()
	targetID := strings.TrimSpace(source.GetSurfaceId())
	if targetID == "" {
		return ProbeRequest{}, false
	}
	request := ProbeRequest{
		ProbeID:                  probeID,
		TargetID:                 targetID,
		ProviderSurfaceBindingID: strings.TrimSpace(surface.GetSurfaceId()),
	}
	if shouldPassSurfaceBaseURL(surface) {
		request.BaseURL = strings.TrimSpace(providerv1.RuntimeBaseURL(runtime))
		request.Protocol = providerv1.RuntimeProtocol(runtime).String()
	}
	return request, true
}

func shouldPassSurfaceBaseURL(surface *providerv1.ProviderSurfaceBinding) bool {
	if surface == nil {
		return false
	}
	runtime := surface.GetRuntime()
	return providerv1.RuntimeKind(runtime) == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API &&
		strings.TrimSpace(providerv1.RuntimeBaseURL(runtime)) != ""
}

// catalogFromModelIDs builds a provider model catalog from discovered model IDs.
// It does inline best-effort binding: if a model ID can be resolved to a canonical
// ModelRef via identity normalization, it is bound immediately.
// Model IDs rejected by the configured ModelIDFilter are excluded from the catalog.
func (m *CatalogMaterializer) catalogFromModelIDs(current *providerv1.ProviderModelCatalog, modelIDs []string, vendorID string) *providerv1.ProviderModelCatalog {
	existingRefs := existingModelRefs(current)
	entries := make([]*providerv1.ProviderModelCatalogEntry, 0, len(modelIDs))
	for _, rawModelID := range modelIDs {
		providerModelID := strings.TrimSpace(rawModelID)
		if providerModelID == "" {
			continue
		}
		if m.modelFilter != nil && !m.modelFilter(providerModelID) {
			continue
		}
		modelRef := existingRefs[providerModelID]
		if modelRef == nil {
			modelRef = resolveModelRef(vendorID, providerModelID)
		}
		entries = append(entries, &providerv1.ProviderModelCatalogEntry{
			ProviderModelId: providerModelID,
			ModelRef:        modelRef,
		})
	}
	return &providerv1.ProviderModelCatalog{
		Models:    entries,
		Source:    providerv1.CatalogSource_CATALOG_SOURCE_MODEL_SERVICE,
		UpdatedAt: timestamppb.Now(),
	}
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

func catalogAlreadyCurrent(current *providerv1.ProviderModelCatalog, modelIDs []string) bool {
	if current.GetSource() != providerv1.CatalogSource_CATALOG_SOURCE_MODEL_SERVICE {
		return false
	}
	currentModels := current.GetModels()
	if len(currentModels) != len(modelIDs) {
		return false
	}
	for index, modelID := range modelIDs {
		if strings.TrimSpace(currentModels[index].GetProviderModelId()) != strings.TrimSpace(modelID) {
			return false
		}
	}
	return true
}

func existingModelRefs(catalog *providerv1.ProviderModelCatalog) map[string]*modelv1.ModelRef {
	out := map[string]*modelv1.ModelRef{}
	for _, item := range catalog.GetModels() {
		modelID := strings.TrimSpace(item.GetProviderModelId())
		if modelID == "" || item.GetModelRef() == nil {
			continue
		}
		out[modelID] = proto.Clone(item.GetModelRef()).(*modelv1.ModelRef)
	}
	return out
}

func catalogOwnerVendorID(surface *providerv1.ProviderSurfaceBinding) string {
	if surface.GetSourceRef().GetKind() != providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR {
		return ""
	}
	return strings.TrimSpace(surface.GetSourceRef().GetId())
}
