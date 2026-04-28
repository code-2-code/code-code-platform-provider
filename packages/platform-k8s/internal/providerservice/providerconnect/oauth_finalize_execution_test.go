package providerconnect

import (
	"context"
	"fmt"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestOAuthFinalizeRuntimeUsesCreatedSurfaceBinding(t *testing.T) {
	providers := &finalizeProviderSurfaceBindingServiceStub{}
	runtime := newProviderConnectOAuthFinalizeRuntime(
		newProviderConnectResources(finalizeCredentialServiceStub{}, providers),
		newProviderConnectQueries(providers, nil, nil),
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

	surface, err := runtime.Finalize(context.Background(), record, &credentialv1.OAuthAuthorizationSessionState{
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
	if got, want := surface.GetSurfaceId(), "codex"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got, want := providers.created.GetSurfaces()[0].GetSourceRef().GetSurfaceId(), "codex"; got != want {
		t.Fatalf("source surface_id = %q, want %q", got, want)
	}
	if got := providers.listCalls; got != 0 {
		t.Fatalf("ListProviderSurfaceBindings() calls = %d, want 0", got)
	}
	if err := providerv1.ValidateProvider(providers.created); err != nil {
		t.Fatalf("ValidateProvider(created) error = %v", err)
	}
}

type finalizeProviderSurfaceBindingServiceStub struct {
	created   *providerv1.Provider
	listCalls int
}

func (s *finalizeProviderSurfaceBindingServiceStub) CreateProvider(
	_ context.Context,
	provider *providerv1.Provider,
) (*ProviderView, error) {
	s.created = provider
	return providerViewFromProvider(provider), nil
}

func (s *finalizeProviderSurfaceBindingServiceStub) ListProviderSurfaceBindings(
	context.Context,
) ([]*ProviderSurfaceBindingView, error) {
	s.listCalls += 1
	return nil, fmt.Errorf("provider surface cache is not ready")
}

type finalizeCredentialServiceStub struct{}

func (finalizeCredentialServiceStub) CreateAPIKey(
	context.Context,
	CredentialAPIKeyCreate,
) (string, error) {
	return "", fmt.Errorf("unexpected api key credential create")
}

func (finalizeCredentialServiceStub) CreateSession(
	context.Context,
	CredentialSessionCreate,
) (string, error) {
	return "", fmt.Errorf("unexpected session credential create")
}

func (finalizeCredentialServiceStub) Delete(context.Context, string) error {
	return nil
}

func providerViewFromProvider(providerResource *providerv1.Provider) *ProviderView {
	if providerResource == nil {
		return nil
	}
	primary := firstProviderSurfaceBindingForTest(providerResource)
	view := &ProviderView{
		ProviderID:           providerResource.GetProviderId(),
		DisplayName:          providerResource.GetDisplayName(),
		VendorID:             providerSurfaceBindingVendorIDForTest(primary),
		ProviderCredentialID: primary.GetProviderCredentialRef().GetProviderCredentialId(),
	}
	for _, surface := range providerResource.GetSurfaces() {
		view.Surfaces = append(view.Surfaces, &ProviderSurfaceBindingView{
			DisplayName:          surface.GetRuntime().GetDisplayName(),
			SurfaceID:            surface.GetSurfaceId(),
			ProviderCredentialID: surface.GetProviderCredentialRef().GetProviderCredentialId(),
			Runtime:              surface.GetRuntime(),
			VendorID:             providerSurfaceBindingVendorIDForTest(surface),
			ProviderID:           providerResource.GetProviderId(),
			ProviderDisplayName:  providerResource.GetDisplayName(),
		})
	}
	return view
}

func firstProviderSurfaceBindingForTest(provider *providerv1.Provider) *providerv1.ProviderSurfaceBinding {
	if provider == nil || len(provider.GetSurfaces()) == 0 {
		return &providerv1.ProviderSurfaceBinding{}
	}
	return provider.GetSurfaces()[0]
}

func providerSurfaceBindingVendorIDForTest(surface *providerv1.ProviderSurfaceBinding) string {
	if surface.GetSourceRef().GetKind() != providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR {
		return ""
	}
	return surface.GetSourceRef().GetId()
}
