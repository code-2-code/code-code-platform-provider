package providerconnect

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

type surfaceModelSet struct {
	bySurfaceID map[string][]*providerv1.ProviderModel
	matched     map[string]struct{}
}

func newSurfaceModelSet(items []*SurfaceModelInput) (*surfaceModelSet, error) {
	set := &surfaceModelSet{
		bySurfaceID: map[string][]*providerv1.ProviderModel{},
		matched:     map[string]struct{}{},
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		surfaceID := strings.TrimSpace(item.SurfaceID)
		if surfaceID == "" {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: surface_id is required for provider models")
		}
		if _, ok := set.bySurfaceID[surfaceID]; ok {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: duplicate provider models for surface %q", surfaceID)
		}
		models := cloneProviderModels(item.Models)
		if err := providerv1.ValidateProviderModels(models); err != nil {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: invalid provider models for surface %q: %v", surfaceID, err)
		}
		set.bySurfaceID[surfaceID] = models
	}
	return set, nil
}

func (s *surfaceModelSet) Models(surfaceID string, fallback []*providerv1.ProviderModel) []*providerv1.ProviderModel {
	if s == nil {
		return cloneProviderModels(fallback)
	}
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return cloneProviderModels(fallback)
	}
	if models, ok := s.bySurfaceID[surfaceID]; ok {
		s.matched[surfaceID] = struct{}{}
		return cloneProviderModels(models)
	}
	return cloneProviderModels(fallback)
}

func (s *surfaceModelSet) ValidateAllMatched() error {
	if s == nil {
		return nil
	}
	for surfaceID := range s.bySurfaceID {
		if _, ok := s.matched[surfaceID]; ok {
			continue
		}
		return domainerror.NewValidation("platformk8s/providerconnect: unknown provider models surface %q", surfaceID)
	}
	return nil
}
