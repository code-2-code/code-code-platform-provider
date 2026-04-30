package providerconnect

import (
	"context"
	"errors"
	"testing"

	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestAPIKeyConnectExecutionAllowsEmptySurfaceCatalog(t *testing.T) {
	t.Parallel()

	target := newConnectTargetWithIDs(
		AddMethodAPIKey,
		"Mistral",
		"mistral",
		"",
		"openai-compatible",
		"credential-mistral",
		"provider-mistral",
		testProviderView("openai-compatible", "openai-compatible"),
	)
	execution := newCustomAPIKeyConnectExecution(target, "credential-mistral")
	var created *providerv1.Provider

	result, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateProvider: func(_ context.Context, provider *providerv1.Provider) (*ProviderView, error) {
			created = provider
			return &ProviderView{ProviderID: provider.GetProviderId()}, nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := result.TargetProviderID, "provider-mistral"; got != want {
		t.Fatalf("target_provider_id = %q, want %q", got, want)
	}
	if got, want := result.Provider.GetProviderId(), "provider-mistral"; got != want {
		t.Fatalf("result provider_id = %q, want %q", got, want)
	}
	if got := len(created.GetRuntime().GetCatalog().GetModels()); got != 0 {
		t.Fatalf("catalog models len = %d, want 0", got)
	}
	if err := providerv1.ValidateProvider(created); err != nil {
		t.Fatalf("ValidateProvider(created) error = %v", err)
	}
}

func TestAPIKeyConnectExecutionReturnsProviderFailure(t *testing.T) {
	t.Parallel()

	execution := newCustomAPIKeyConnectExecution(testAPIKeyTarget(), "credential-openai")

	_, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateProvider: func(context.Context, *providerv1.Provider) (*ProviderView, error) {
			return nil, errors.New("provider store failed")
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want provider failure")
	}
}

func TestAPIKeyConnectExecutionRequiresCredentialID(t *testing.T) {
	t.Parallel()

	execution := newCustomAPIKeyConnectExecution(testAPIKeyTarget(), "")
	_, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateProvider: func(context.Context, *providerv1.Provider) (*ProviderView, error) {
			return &ProviderView{ProviderID: "provider-openai"}, nil
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want credential_id validation error")
	}
}

func TestAPIKeyConnectExecutionReturnsNilProviderFailure(t *testing.T) {
	t.Parallel()

	execution := newCustomAPIKeyConnectExecution(testAPIKeyTarget(), "credential-openai")

	_, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateProvider: func(context.Context, *providerv1.Provider) (*ProviderView, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want nil provider failure")
	}
}

func testAPIKeyTarget() *connectTarget {
	return newConnectTargetWithIDs(
		AddMethodAPIKey,
		"OpenAI",
		"openai",
		"",
		"openai-compatible",
		"credential-openai",
		"provider-openai",
		testProviderView("openai-compatible", "openai-compatible"),
	)
}
