package providerconnect

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type surfaceCatalogSet struct {
	bySurfaceID map[string]*providerv1.ProviderModelCatalog
	matched     map[string]struct{}
}

func newSurfaceCatalogSet(items []*ProviderSurfaceBindingModelCatalogInput) (*surfaceCatalogSet, error) {
	set := &surfaceCatalogSet{
		bySurfaceID: map[string]*providerv1.ProviderModelCatalog{},
		matched:     map[string]struct{}{},
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		surfaceID := strings.TrimSpace(item.SurfaceID)
		if surfaceID == "" {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: surface_id is required for surface model catalog")
		}
		if _, ok := set.bySurfaceID[surfaceID]; ok {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: duplicate surface catalog %q", surfaceID)
		}
		catalog := &providerv1.ProviderModelCatalog{
			Models: cloneProviderSurfaceBindingModels(item.Models),
			Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
		}
		if err := validateCatalogModels(catalog.GetModels()); err != nil {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: invalid surface model catalog %q: %v", surfaceID, err)
		}
		set.bySurfaceID[surfaceID] = catalog
	}
	return set, nil
}

func (s *surfaceCatalogSet) Override(surfaceID string, fallback *providerv1.ProviderModelCatalog) *providerv1.ProviderModelCatalog {
	if s == nil {
		return cloneSurfaceModelCatalog(fallback)
	}
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return cloneSurfaceModelCatalog(fallback)
	}
	if catalog := s.bySurfaceID[surfaceID]; catalog != nil {
		s.matched[surfaceID] = struct{}{}
		return cloneSurfaceModelCatalog(catalog)
	}
	return cloneSurfaceModelCatalog(fallback)
}

func (s *surfaceCatalogSet) ValidateAllMatched() error {
	if s == nil {
		return nil
	}
	for surfaceID := range s.bySurfaceID {
		if _, ok := s.matched[surfaceID]; ok {
			continue
		}
		return domainerror.NewValidation("platformk8s/providerconnect: unknown surface catalog %q", surfaceID)
	}
	return nil
}

func cloneSurfaceModelCatalog(catalog *providerv1.ProviderModelCatalog) *providerv1.ProviderModelCatalog {
	if catalog == nil {
		return nil
	}
	return proto.Clone(catalog).(*providerv1.ProviderModelCatalog)
}

func validateCatalogModels(items []*providerv1.ProviderModelCatalogEntry) error {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item == nil {
			return domainerror.NewValidation("platformk8s/providerconnect: surface model entry is nil")
		}
		providerModelID := strings.TrimSpace(item.GetProviderModelId())
		if providerModelID == "" {
			return domainerror.NewValidation("platformk8s/providerconnect: surface model provider_model_id is required")
		}
		if _, ok := seen[providerModelID]; ok {
			return domainerror.NewValidation("platformk8s/providerconnect: duplicate surface model %q", providerModelID)
		}
		seen[providerModelID] = struct{}{}
	}
	return nil
}
