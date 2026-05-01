package providercatalogs

import (
	"context"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestMaterializeProviderUsesSourceEndpointTemplateID(t *testing.T) {
	t.Parallel()

	probe := &probeStub{
		modelIDs: []string{"MiniMax-Text-01"},
	}
	materializer := NewCatalogMaterializer(probe, nil, nil, surfaceReaderStub{
		"minimax-6e6300": apiSurface("minimax-6e6300", "surface.openai-compatible", apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, "https://api.minimaxi.com/v1"),
	})

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "minimax-account",
		DisplayName: "MiniMax",
		SurfaceId:   "minimax-6e6300",
		ProviderCredentialRef: &providerv1.ProviderCredentialRef{
			ProviderCredentialId: "credential-minimax",
		},
	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	if got, want := probe.last.TargetID, "minimax-account"; got != want {
		t.Fatalf("catalog target id = %q, want %q", got, want)
	}
	if got, want := probe.last.ProbeID, "surface.openai-compatible"; got != want {
		t.Fatalf("catalog probe id = %q, want %q", got, want)
	}
	if got, want := probe.last.ProviderCredentialID, "credential-minimax"; got != want {
		t.Fatalf("provider credential id = %q, want %q", got, want)
	}
	if got, want := provider.GetModels()[0].GetProviderModelId(), "MiniMax-Text-01"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}

func TestMaterializeProviderAppliesFilterBeforeCurrentCheck(t *testing.T) {
	t.Parallel()

	probe := &probeStub{
		modelIDs: []string{"gemini-2.5-pro", "imagen-4.0-generate-preview"},
	}
	materializer := NewCatalogMaterializer(probe, nil, func(input ModelIDFilterInput) bool {
		return input.ProviderModelID != "imagen-4.0-generate-preview"
	}, surfaceReaderStub{
		"gemini": apiSurface("gemini", "gemini", apiprotocolv1.Protocol_PROTOCOL_GEMINI, "https://generativelanguage.googleapis.com/v1beta"),
	})

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "google-account",
		DisplayName: "Google",
		SurfaceId:   "gemini",
		Models: []*providerv1.ProviderModel{
			{ProviderModelId: "gemini-2.5-pro"},
			{ProviderModelId: "imagen-4.0-generate-preview"},
		},
	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	models := provider.GetModels()
	if got, want := len(models), 1; got != want {
		t.Fatalf("len(models) = %d, want %d", got, want)
	}
	if got, want := models[0].GetProviderModelId(), "gemini-2.5-pro"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}

func TestMaterializeProviderUsesCustomAPIKeyOpenAIModelProbe(t *testing.T) {
	t.Parallel()

	probe := &probeStub{
		modelIDs: []string{"custom-model"},
	}
	materializer := NewCatalogMaterializer(probe, nil, nil, surfaceReaderStub{})

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "custom-account",
		DisplayName: "Custom",
		SurfaceId:   "custom.api",
		ProviderCredentialRef: &providerv1.ProviderCredentialRef{
			ProviderCredentialId: "credential-custom",
		},
		CustomApiKeySurface: &providerv1.CustomAPIKeySurface{
			BaseUrl:  "https://api.custom.example/v1",
			Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		},
	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	if got, want := probe.last.ProbeID, "surface.openai-compatible"; got != want {
		t.Fatalf("catalog probe id = %q, want %q", got, want)
	}
	if got, want := probe.last.BaseURL, "https://api.custom.example/v1"; got != want {
		t.Fatalf("catalog base_url = %q, want %q", got, want)
	}
	if got, want := provider.GetModels()[0].GetProviderModelId(), "custom-model"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}

func TestMaterializeProviderSkipsCustomAPIKeyNonOpenAIModelProbe(t *testing.T) {
	t.Parallel()

	probe := &probeStub{}
	materializer := NewCatalogMaterializer(probe, nil, nil, surfaceReaderStub{})

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "custom-account",
		DisplayName: "Custom",
		SurfaceId:   "custom.api",
		CustomApiKeySurface: &providerv1.CustomAPIKeySurface{
			BaseUrl:  "https://api.custom.example",
			Protocol: apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC,
		},
	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	if probe.last.ProbeID != "" {
		t.Fatalf("catalog probe id = %q, want empty", probe.last.ProbeID)
	}
	if len(provider.GetModels()) != 0 {
		t.Fatalf("models len = %d, want 0", len(provider.GetModels()))
	}
}

func TestMaterializeProviderUsesSurfaceProbeForParameterizedVendorSurface(t *testing.T) {
	t.Parallel()

	probe := &probeStub{
		modelIDs: []string{"@cf/meta/llama-3.1-8b-instruct"},
	}
	materializer := NewCatalogMaterializer(probe, nil, nil, surfaceReaderStub{
		"cloudflare-workers-ai": apiSurface(
			"cloudflare-workers-ai",
			"surface.cloudflare-workers-ai",
			apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
			"https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1",
		),
	})

	provider, err := materializer.MaterializeProvider(context.Background(), &providerv1.Provider{
		ProviderId:  "cloudflare-workers-ai",
		DisplayName: "Cloudflare Workers AI",
		SurfaceId:   "cloudflare-workers-ai",
		ProviderCredentialRef: &providerv1.ProviderCredentialRef{
			ProviderCredentialId: "credential-cloudflare",
		},
		CustomApiKeySurface: &providerv1.CustomAPIKeySurface{
			BaseUrl:  "https://api.cloudflare.com/client/v4/accounts/04d289f3ff972711c415793f0b7da61d/ai/v1",
			Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		},
	})
	if err != nil {
		t.Fatalf("MaterializeProvider() error = %v", err)
	}
	if got, want := probe.last.ProbeID, "surface.cloudflare-workers-ai"; got != want {
		t.Fatalf("catalog probe id = %q, want %q", got, want)
	}
	if got, want := probe.last.Operation.GetPath(), "models/search"; got != want {
		t.Fatalf("catalog operation path = %q, want %q", got, want)
	}
	if got, want := probe.last.Operation.GetBaseUrl(), "https://api.cloudflare.com/client/v4/accounts/04d289f3ff972711c415793f0b7da61d/ai"; got != want {
		t.Fatalf("catalog operation base_url = %q, want %q", got, want)
	}
	if got, want := provider.GetModels()[0].GetProviderModelId(), "@cf/meta/llama-3.1-8b-instruct"; got != want {
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

type surfaceReaderStub map[string]*supportv1.Surface

func (s surfaceReaderStub) Get(_ context.Context, surfaceID string) (*supportv1.Surface, error) {
	return s[surfaceID], nil
}

func apiSurface(surfaceID, probeID string, protocol apiprotocolv1.Protocol, baseURL string) *supportv1.Surface {
	return &supportv1.Surface{
		SurfaceId:           surfaceID,
		ModelCatalogProbeId: probeID,
		Spec: &supportv1.Surface_Api{
			Api: &supportv1.ApiSurface{ApiEndpoints: []*supportv1.ApiEndpoint{{
				Protocol: protocol,
				BaseUrl:  baseURL,
			}}},
		},
	}
}
