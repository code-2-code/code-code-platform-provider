package providersurfaces

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
)

func TestServiceListReturnsBuiltinSurfaces(t *testing.T) {
	t.Parallel()

	service, err := NewService()
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	items, err := service.List(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("List() len = %d, want at least 3", len(items))
	}
	openAICompatibleSurface, err := service.Get(t.Context(), "openai-compatible")
	if err != nil {
		t.Fatalf("Get() openai-compatible error = %v", err)
	}
	if got, want := openAICompatibleSurface.GetProductInfoId(), "openai"; got != want {
		t.Fatalf("product_info_id = %q, want %q", got, want)
	}
	if got, want := openAICompatibleSurface.GetApi().GetApiEndpoints()[0].GetProtocol(), apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE; got != want {
		t.Fatalf("endpoint protocol = %v, want %v", got, want)
	}
	geminiSurface, err := service.Get(t.Context(), "google-gemini")
	if err != nil {
		t.Fatalf("Get() gemini error = %v", err)
	}
	if got, want := geminiSurface.GetModelCatalogProbeId(), "google-models"; got != want {
		t.Fatalf("model_catalog_probe_id = %q, want %q", got, want)
	}
	if got, want := geminiSurface.GetQuotaProbeId(), "google-aistudio-quotas"; got != want {
		t.Fatalf("quota_probe_id = %q, want %q", got, want)
	}
	anthropicSurface, err := service.Get(t.Context(), "minimax-anthropic")
	if err != nil {
		t.Fatalf("Get() vendor error = %v", err)
	}
	if got, want := anthropicSurface.GetModelCatalogProbeId(), "surface.minimax-anthropic"; got != want {
		t.Fatalf("model_catalog_probe_id = %q, want %q", got, want)
	}
}
