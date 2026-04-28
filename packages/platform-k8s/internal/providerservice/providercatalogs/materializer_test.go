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
		Surfaces: []*providerv1.ProviderSurfaceBinding{{
			SurfaceId: "minimax-6e6300",
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
			SourceRef: &providerv1.ProviderSurfaceSourceRef{
				Kind:      providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR,
				Id:        "minimax",
				SurfaceId: "minimax-openai-compatible",
			},
		}},
	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	if got, want := probe.last.TargetID, "minimax-openai-compatible"; got != want {
		t.Fatalf("catalog target id = %q, want %q", got, want)
	}
	if got, want := probe.last.ProbeID, "surface.openai-compatible"; got != want {
		t.Fatalf("catalog probe id = %q, want %q", got, want)
	}
	if got, want := provider.GetSurfaces()[0].GetRuntime().GetCatalog().GetModels()[0].GetProviderModelId(), "MiniMax-Text-01"; got != want {
		t.Fatalf("catalog provider_model_id = %q, want %q", got, want)
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
