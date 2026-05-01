package providercatalogs

import (
	"context"
	"net/http"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
)

func TestMaterializerProbePassesResolvedAuthHeaders(t *testing.T) {
	t.Parallel()

	executor := &fakeCatalogModelIDExecutor{modelIDs: []string{"mistral-small-latest"}}
	headerResolver := &fakeCatalogProbeHeaderResolver{
		headers: http.Header{"Authorization": []string{"Bearer token-1"}},
	}
	probe := NewMaterializerProbe(executor, headerResolver)

	modelIDs, err := probe.ProbeModelIDs(context.Background(), ProbeRequest{
		ProbeID:              "surface.openai-compatible",
		TargetID:             "mistral-openai-compatible",
		BaseURL:              "https://api.mistral.ai/v1",
		Protocol:             apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		SurfaceID:            "mistral-openai-compatible",
		ProviderCredentialID: "credential-mistral",
	})
	if err != nil {
		t.Fatalf("ProbeModelIDs() error = %v", err)
	}
	if got, want := modelIDs[0], "mistral-small-latest"; got != want {
		t.Fatalf("model id = %q, want %q", got, want)
	}
	if got, want := headerResolver.last.CredentialID, "credential-mistral"; got != want {
		t.Fatalf("resolved credential id = %q, want %q", got, want)
	}
	if got, want := executor.last.Headers.Get("Authorization"), "Bearer token-1"; got != want {
		t.Fatalf("authorization = %q, want %q", got, want)
	}
}

type fakeCatalogModelIDExecutor struct {
	last     CatalogProbeRequest
	modelIDs []string
	err      error
}

func (e *fakeCatalogModelIDExecutor) ProbeModelIDs(_ context.Context, request CatalogProbeRequest) ([]string, error) {
	e.last = request
	return e.modelIDs, e.err
}
