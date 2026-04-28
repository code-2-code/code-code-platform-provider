package anthropicprovider

import (
	"context"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	modelv1 "code-code.internal/go-contract/model/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestRuntimeListModelsUsesConfiguredSurfaceCatalog(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	runtime, err := provider.NewRuntime(
		&providerv1.ProviderSurfaceBinding{
			SurfaceId: "instance-1",
			Runtime: &providerv1.ProviderSurfaceRuntime{
				DisplayName: "instance-1",
				Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
				Access: &providerv1.ProviderSurfaceRuntime_Api{
					Api: &providerv1.ProviderAPISurfaceRuntime{
						Protocol: apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC,
						BaseUrl:  "https://api.example.com/anthropic",
					},
				},
				Catalog: &providerv1.ProviderModelCatalog{
					Models: []*providerv1.ProviderModelCatalogEntry{{
						ProviderModelId: "model-core",
						ModelRef: &modelv1.ModelRef{
							ModelId: "shared-model",
						},
					}},
					Source: providerv1.CatalogSource_CATALOG_SOURCE_FALLBACK_CONFIG,
				},
			},
		},
		apiKeyCredential("test-key"),
	)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	catalog, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := len(catalog.GetModels()), 1; got != want {
		t.Fatalf("catalog model count = %d, want %d", got, want)
	}
	if got, want := catalog.GetModels()[0].GetProviderModelId(), "model-core"; got != want {
		t.Fatalf("catalog provider model id = %q, want %q", got, want)
	}
	if got, want := catalog.GetModels()[0].GetModelRef().GetModelId(), "shared-model"; got != want {
		t.Fatalf("catalog model ref id = %q, want %q", got, want)
	}
}

func apiKeyCredential(apiKey string) *credentialv1.ResolvedCredential {
	return &credentialv1.ResolvedCredential{
		CredentialId: "sample-credential",
		Kind:         credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY,
		Material: &credentialv1.ResolvedCredential_ApiKey{
			ApiKey: &credentialv1.ApiKeyCredential{ApiKey: apiKey},
		},
	}
}
