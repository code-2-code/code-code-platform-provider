package providerconnect

import (
	"context"
	"errors"
	"strings"
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
		testProviderSurfaceBinding("openai-compatible", "openai-compatible"),
	)
	execution := newCustomAPIKeyConnectExecution(target, "sk-mistral")
	var created *providerv1.Provider

	result, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateCredential: func(context.Context, CredentialAPIKeyCreate) (string, error) {
			return "credential-mistral", nil
		},
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
	surface := created.GetSurfaces()[0]
	if got := len(surface.GetRuntime().GetCatalog().GetModels()); got != 0 {
		t.Fatalf("catalog models len = %d, want 0", got)
	}
	if err := providerv1.ValidateProvider(created); err != nil {
		t.Fatalf("ValidateProvider(created) error = %v", err)
	}
}

func TestAPIKeyConnectExecutionRollsBackCredentialOnProviderFailure(t *testing.T) {
	t.Parallel()

	execution := newCustomAPIKeyConnectExecution(testAPIKeyTarget(), "sk-openai")
	var deletedCredentialID string

	_, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateCredential: func(context.Context, CredentialAPIKeyCreate) (string, error) {
			return "credential-openai", nil
		},
		CreateProvider: func(context.Context, *providerv1.Provider) (*ProviderView, error) {
			return nil, errors.New("provider store failed")
		},
		DeleteCredential: func(_ context.Context, credentialID string) error {
			deletedCredentialID = credentialID
			return nil
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want provider failure")
	}
	if got, want := deletedCredentialID, "credential-openai"; got != want {
		t.Fatalf("deleted credential id = %q, want %q", got, want)
	}
}

func TestAPIKeyConnectExecutionReturnsRollbackFailure(t *testing.T) {
	t.Parallel()

	execution := newCustomAPIKeyConnectExecution(testAPIKeyTarget(), "sk-openai")
	_, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateCredential: func(context.Context, CredentialAPIKeyCreate) (string, error) {
			return "credential-openai", nil
		},
		CreateProvider: func(context.Context, *providerv1.Provider) (*ProviderView, error) {
			return nil, errors.New("provider store failed")
		},
		DeleteCredential: func(context.Context, string) error {
			return errors.New("delete denied")
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want rollback failure")
	}
	if !strings.Contains(err.Error(), "rollback credential") {
		t.Fatalf("Execute() error = %q, want rollback detail", err.Error())
	}
}

func TestAPIKeyConnectExecutionRollsBackCredentialOnNilProviderResult(t *testing.T) {
	t.Parallel()

	execution := newCustomAPIKeyConnectExecution(testAPIKeyTarget(), "sk-openai")
	var deletedCredentialID string

	_, err := execution.Execute(context.Background(), apiKeyConnectRuntime{
		CreateCredential: func(context.Context, CredentialAPIKeyCreate) (string, error) {
			return "credential-openai", nil
		},
		CreateProvider: func(context.Context, *providerv1.Provider) (*ProviderView, error) {
			return nil, nil
		},
		DeleteCredential: func(_ context.Context, credentialID string) error {
			deletedCredentialID = credentialID
			return nil
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want nil provider failure")
	}
	if got, want := deletedCredentialID, "credential-openai"; got != want {
		t.Fatalf("deleted credential id = %q, want %q", got, want)
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
		testProviderSurfaceBinding("openai-compatible", "openai-compatible"),
	)
}
