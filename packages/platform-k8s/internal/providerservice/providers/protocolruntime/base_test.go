package protocolruntime

import (
	"context"
	"testing"
	"time"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	modelv1 "code-code.internal/go-contract/model/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestBaseRuntimeListModelsBuildsCatalogFromConfiguredSurfaceModels(t *testing.T) {
	t.Parallel()

	runtime := &BaseRuntime{
		Surface: &providerv1.ProviderSurfaceBinding{
			SurfaceId: "instance-1",
			Runtime: &providerv1.ProviderSurfaceRuntime{
				DisplayName: "instance-1",
				Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
				Access: &providerv1.ProviderSurfaceRuntime_Api{
					Api: &providerv1.ProviderAPISurfaceRuntime{
						Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
						BaseUrl:  "https://api.example.com/v1",
					},
				},
				Catalog: &providerv1.ProviderModelCatalog{
					Models: []*providerv1.ProviderModelCatalogEntry{
						{ProviderModelId: "model-a", ModelRef: &modelv1.ModelRef{ModelId: "model-a", VendorId: "openai"}},
						{ProviderModelId: "model-b", ModelRef: &modelv1.ModelRef{ModelId: "model-b", VendorId: "openai"}},
					},
					Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
				},
			},
		},
		Now: func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}

	catalog, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := len(catalog.GetModels()), 2; got != want {
		t.Fatalf("catalog models len = %d, want %d", got, want)
	}
	if got, want := catalog.GetModels()[1].GetProviderModelId(), "model-b"; got != want {
		t.Fatalf("provider model id = %q, want %q", got, want)
	}
}

func TestBaseRuntimeListModelsAllowsEmptySurfaceCatalog(t *testing.T) {
	t.Parallel()

	runtime := &BaseRuntime{
		Surface: &providerv1.ProviderSurfaceBinding{
			SurfaceId: "instance-1",
			Runtime: &providerv1.ProviderSurfaceRuntime{
				DisplayName: "instance-1",
				Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
				Access: &providerv1.ProviderSurfaceRuntime_Api{
					Api: &providerv1.ProviderAPISurfaceRuntime{
						Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
						BaseUrl:  "https://api.example.com/v1",
					},
				},
				Catalog: &providerv1.ProviderModelCatalog{},
			},
		},
		Now: func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}

	catalog, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got := len(catalog.GetModels()); got != 0 {
		t.Fatalf("catalog models len = %d, want 0", got)
	}
}
