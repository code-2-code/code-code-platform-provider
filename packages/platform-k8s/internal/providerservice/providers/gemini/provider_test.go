package geminiprovider

import (
	"context"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestRuntimeListModelsUsesConfiguredSurfaceCatalog(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	runtime, err := provider.NewRuntime(
		&providerv1.ProviderSurfaceBinding{
			SurfaceId: "gemini",
			Runtime: &providerv1.ProviderSurfaceRuntime{
				DisplayName: "Google Gemini",
				Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
				Access: &providerv1.ProviderSurfaceRuntime_Api{
					Api: &providerv1.ProviderAPISurfaceRuntime{
						Protocol: apiprotocolv1.Protocol_PROTOCOL_GEMINI,
						BaseUrl:  "https://generativelanguage.googleapis.com/v1beta",
					},
				},
				Catalog: &providerv1.ProviderModelCatalog{
					Models: []*providerv1.ProviderModelCatalogEntry{{
						ProviderModelId: "gemini-2.5-flash",
					}},
					Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
				},
			},
		},
		&credentialv1.ResolvedCredential{
			Kind: credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY,
			Material: &credentialv1.ResolvedCredential_ApiKey{
				ApiKey: &credentialv1.ApiKeyCredential{ApiKey: "test-key"},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	catalog, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := catalog.GetModels()[0].GetProviderModelId(), "gemini-2.5-flash"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}
