package providerconnect

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestNewSurfaceCatalogSetRejectsDuplicateSurfaceCatalog(t *testing.T) {
	_, err := newSurfaceCatalogSet([]*ProviderSurfaceBindingModelCatalogInput{
		{SurfaceID: "openai-compatible", Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}}},
		{SurfaceID: "openai-compatible", Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1-mini"}}},
	})
	if err == nil {
		t.Fatal("newSurfaceCatalogSet() error = nil, want validation error")
	}
}

func TestSurfaceCatalogSetOverrideMarksCatalogMatched(t *testing.T) {
	set, err := newSurfaceCatalogSet([]*ProviderSurfaceBindingModelCatalogInput{{
		SurfaceID: "openai-compatible",
		Models: []*providerv1.ProviderModelCatalogEntry{
			{ProviderModelId: "gpt-4.1"},
			{ProviderModelId: "gpt-4.1-mini"},
		},
	}})
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	catalog := set.Override(apiSurfaceIDForProtocol(apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE), nil)
	if got, want := catalog.GetModels()[1].GetProviderModelId(), "gpt-4.1-mini"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
	if err := set.ValidateAllMatched(); err != nil {
		t.Fatalf("ValidateAllMatched() error = %v", err)
	}
}

func TestNewSurfaceCatalogSetAllowsEmptySurfaceCatalog(t *testing.T) {
	set, err := newSurfaceCatalogSet([]*ProviderSurfaceBindingModelCatalogInput{{
		SurfaceID: "mistral-openai-compatible",
	}})
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	catalog := set.Override("mistral-openai-compatible", nil)
	if got := len(catalog.GetModels()); got != 0 {
		t.Fatalf("len(models) = %d, want 0", got)
	}
	if err := set.ValidateAllMatched(); err != nil {
		t.Fatalf("ValidateAllMatched() error = %v", err)
	}
}

func TestSurfaceCatalogSetValidateAllMatchedRejectsUnknownSurface(t *testing.T) {
	set, err := newSurfaceCatalogSet([]*ProviderSurfaceBindingModelCatalogInput{{
		SurfaceID: "unknown-surface",
		Models:    []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
	}})
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	if err := set.ValidateAllMatched(); err == nil {
		t.Fatal("ValidateAllMatched() error = nil, want validation error")
	}
}
