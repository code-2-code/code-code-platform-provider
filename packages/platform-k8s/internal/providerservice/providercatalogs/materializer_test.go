package providercatalogs

import (
	"context"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestMaterializeProviderUsesSourceEndpointTemplateID(t *testing.T) {
	t.Parallel()

	probe := &probeStub{
		modelIDs: []string{"MiniMax-Text-01"},
	}
	materializer := NewCatalogMaterializer(probe, nil, nil)

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "minimax-account",
		DisplayName: "MiniMax",
		SurfaceId:   "minimax-6e6300",
		Runtime: &providerv1.ProviderSurfaceRuntime{
			DisplayName:         "MiniMax OpenAI Compatible",
			Origin:              providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
			ModelCatalogProbeId: "surface.openai-compatible",
			Access: &providerv1.ProviderSurfaceRuntime_Api{
				Api: &providerv1.ProviderAPISurfaceRuntime{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
					BaseUrl:  "https://api.minimaxi.com/v1",
				},
			},
		},

	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	if got, want := probe.last.TargetID, "minimax-6e6300"; got != want {
		t.Fatalf("catalog target id = %q, want %q", got, want)
	}
	if got, want := probe.last.ProbeID, "surface.openai-compatible"; got != want {
		t.Fatalf("catalog probe id = %q, want %q", got, want)
	}
	if got, want := provider.GetRuntime().GetCatalog().GetModels()[0].GetProviderModelId(), "MiniMax-Text-01"; got != want {
		t.Fatalf("catalog provider_model_id = %q, want %q", got, want)
	}
}

func TestMaterializeProviderAppliesFilterBeforeCurrentCheck(t *testing.T) {
	t.Parallel()

	probe := &probeStub{
		modelIDs: []string{"gemini-2.5-pro", "imagen-4.0-generate-preview"},
	}
	materializer := NewCatalogMaterializer(probe, nil, func(input ModelIDFilterInput) bool {
		return input.ProviderModelID != "imagen-4.0-generate-preview"
	})

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "google-account",
		DisplayName: "Google",
		SurfaceId:   "gemini",
		Runtime: &providerv1.ProviderSurfaceRuntime{
			DisplayName:         "Google Gemini",
			Origin:              providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
			ModelCatalogProbeId: "gemini",
			Catalog: &providerv1.ProviderModelCatalog{
				Source: providerv1.CatalogSource_CATALOG_SOURCE_PROVIDER_DISCOVERY,
				Models: []*providerv1.ProviderModelCatalogEntry{
					{ProviderModelId: "gemini-2.5-pro"},
					{ProviderModelId: "imagen-4.0-generate-preview"},
				},
			},
			Access: &providerv1.ProviderSurfaceRuntime_Api{
				Api: &providerv1.ProviderAPISurfaceRuntime{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_GEMINI,
					BaseUrl:  "https://generativelanguage.googleapis.com/v1beta",
				},
			},
		},

	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	models := provider.GetRuntime().GetCatalog().GetModels()
	if got, want := len(models), 1; got != want {
		t.Fatalf("len(models) = %d, want %d", got, want)
	}
	if got, want := models[0].GetProviderModelId(), "gemini-2.5-pro"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}

type probeStub struct {
	last     ProbeRequest
	modelIDs []string
}

func (s *probeStub) ProbeModelIDs(_ context.Context, request ProbeRequest) ([]string, error) {
	s.last = request
	return s.modelIDs, nil
}
