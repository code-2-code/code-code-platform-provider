package anthropicprovider

import (
	"context"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	modelv1 "code-code.internal/go-contract/model/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestRuntimeListModelsUsesConfiguredSurfaceCatalog(t *testing.T) {
	t.Parallel()

	provider := NewProvider()
	runtime, err := provider.NewRuntime(
		&providerv1.Provider{
			ProviderId:  "provider-1",
			DisplayName: "Provider 1",
			SurfaceId:   surfaceID,
			Models: []*providerv1.ProviderModel{{
				ProviderModelId: "model-core",
				ModelRef: &modelv1.ModelRef{
					ModelId: "shared-model",
				},
			}},
		},
		apiKeyCredential("test-key"),
	)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	models, err := runtime.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if got, want := len(models), 1; got != want {
		t.Fatalf("model count = %d, want %d", got, want)
	}
	if got, want := models[0].GetProviderModelId(), "model-core"; got != want {
		t.Fatalf("provider model id = %q, want %q", got, want)
	}
	if got, want := models[0].GetModelRef().GetModelId(), "shared-model"; got != want {
		t.Fatalf("model ref id = %q, want %q", got, want)
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
