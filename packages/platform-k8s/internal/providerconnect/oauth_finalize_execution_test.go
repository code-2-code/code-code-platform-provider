package providerconnect

import (
	"context"
	"fmt"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestOAuthFinalizeRuntimeUsesCreatedProvider(t *testing.T) {
	providers := &finalizeProviderServiceStub{}
	runtime := newProviderConnectOAuthFinalizeRuntime(
		newProviderConnectResources(providers),
		newProviderConnectQueries(providers, nil),
		nil,
		nil,
	)
	record, err := newSessionRecord(
		"session-1",
		testCLIOAuthSessionTarget("codex"),
		&credentialv1.OAuthAuthorizationSessionStatus{},
	)
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}

	provider, err := runtime.Finalize(context.Background(), record, &credentialv1.OAuthAuthorizationSessionState{
		Spec: &credentialv1.OAuthAuthorizationSessionSpec{
			SessionId:          "session-1",
			TargetCredentialId: "credential-imported",
		},
		Status: &credentialv1.OAuthAuthorizationSessionStatus{
			Phase: credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED,
			ImportedCredential: &credentialv1.ImportedCredentialSummary{
				CredentialId: "credential-imported",
			},
		},
	})
	if err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if got, want := provider.GetSurfaceId(), "codex"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}

	if got := providers.listCalls; got != 0 {
		t.Fatalf("ListProviders() calls = %d, want 0", got)
	}
	if err := providerv1.ValidateProvider(providers.created); err != nil {
		t.Fatalf("ValidateProvider(created) error = %v", err)
	}
}

type finalizeProviderServiceStub struct {
	created   *providerv1.Provider
	listCalls int
}

func (s *finalizeProviderServiceStub) CreateProvider(
	_ context.Context,
	provider *providerv1.Provider,
) (*ProviderView, error) {
	s.created = provider
	return providerViewFromProvider(provider), nil
}

func (s *finalizeProviderServiceStub) List(
	context.Context,
) ([]*ProviderView, error) {
	s.listCalls += 1
	return nil, fmt.Errorf("provider cache is not ready")
}

func (s *finalizeProviderServiceStub) Get(context.Context, string) (*ProviderView, error) {
	return nil, fmt.Errorf("not implemented")
}

func providerViewFromProvider(providerResource *providerv1.Provider) *ProviderView {
	if providerResource == nil {
		return nil
	}
	return &ProviderView{
		ProviderID:           providerResource.GetProviderId(),
		DisplayName:          providerResource.GetDisplayName(),
		VendorID:             providerVendorIDForTest(providerResource),
		ProviderCredentialID: providerResource.GetProviderCredentialRef().GetProviderCredentialId(),
		SurfaceID:            providerResource.GetSurfaceId(),
		Runtime:              providerResource.GetRuntime(),
		ProviderDisplayName:  providerResource.GetDisplayName(),
	}
}

func firstProviderForTest(provider *providerv1.Provider) *providerv1.Provider {
	return provider
}

func providerVendorIDForTest(provider *providerv1.Provider) string {
	return provider.GetSurfaceId() // Just return surfaceID since VendorId logic is gone
}
