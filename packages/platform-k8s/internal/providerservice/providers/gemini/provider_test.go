package geminiprovider

import (
	"context"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestRuntimeListModelsUsesConfiguredSurfaceCatalog(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	runtime, err := provider.NewRuntime(
		&providerv1.Provider{
			ProviderId:  "provider-gemini",
			DisplayName: "Google Gemini",
			SurfaceId:   "gemini",
			Models: []*providerv1.ProviderModel{{
				ProviderModelId: "gemini-2.5-flash",
			}},
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

	models, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := models[0].GetProviderModelId(), "gemini-2.5-flash"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}
