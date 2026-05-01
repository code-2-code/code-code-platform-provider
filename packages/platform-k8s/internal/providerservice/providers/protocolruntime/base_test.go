package protocolruntime

import (
	"context"
	"testing"
	"time"

	modelv1 "code-code.internal/go-contract/model/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestBaseRuntimeListModelsBuildsCatalogFromConfiguredSurfaceModels(t *testing.T) {
	t.Parallel()

	runtime := &BaseRuntime{
		Provider: &providerv1.Provider{
			ProviderId:  "provider-1",
			DisplayName: "Provider 1",
			SurfaceId:   "instance-1",
			Models: []*providerv1.ProviderModel{
				{ProviderModelId: "model-a", ModelRef: &modelv1.ModelRef{ModelId: "model-a", VendorId: "openai"}},
				{ProviderModelId: "model-b", ModelRef: &modelv1.ModelRef{ModelId: "model-b", VendorId: "openai"}},
			},
		},
		Now: func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}

	models, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := len(models), 2; got != want {
		t.Fatalf("models len = %d, want %d", got, want)
	}
	if got, want := models[1].GetProviderModelId(), "model-b"; got != want {
		t.Fatalf("provider model id = %q, want %q", got, want)
	}
}

func TestBaseRuntimeListModelsAllowsEmptySurfaceCatalog(t *testing.T) {
	t.Parallel()

	runtime := &BaseRuntime{
		Provider: &providerv1.Provider{
			ProviderId:  "provider-1",
			DisplayName: "Provider 1",
			SurfaceId:   "instance-1",
		},
		Now: func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}

	models, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got := len(models); got != 0 {
		t.Fatalf("models len = %d, want 0", got)
	}
}
