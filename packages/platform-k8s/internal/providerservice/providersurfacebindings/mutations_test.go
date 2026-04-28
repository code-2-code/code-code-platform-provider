package providersurfacebindings

import (
	"errors"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestCreateProviderSurfaceBindingAddsSurfaceBindingToProviderAggregate(t *testing.T) {
	service, store := newServiceForTest()
	provider := testProvider("provider-account-1")
	provider.Surfaces = []*providerv1.ProviderSurfaceBinding{
		testSurfaceBinding("provider-account-1", "existing-surface", "https://existing.example.com/v1", providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL),
	}
	if _, err := service.CreateProvider(t.Context(), provider); err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	_, err := service.CreateProviderSurfaceBinding(t.Context(), "provider-account-1", testSurfaceBinding("provider-account-1", "sample-openai-compatible", "https://api.example.com/v1", providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL))
	if err != nil {
		t.Fatalf("CreateProviderSurfaceBinding() error = %v", err)
	}

	materialized, err := store.Get(t.Context(), "provider-account-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := len(materialized.GetSurfaces()), 2; got != want {
		t.Fatalf("surfaces len = %d, want %d", got, want)
	}
	if got, want := materialized.GetSurfaces()[1].GetSurfaceId(), "sample-openai-compatible"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
}

func TestUpdateProviderSurfaceBindingRejectsDerivedSurfaceBindingMutation(t *testing.T) {
	service, _ := newServiceForTest()
	provider := testProvider("provider-account-2")
	provider.Surfaces = []*providerv1.ProviderSurfaceBinding{
		testSurfaceBinding("provider-account-2", "derived-surface", "https://openrouter.ai/api/v1", providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED),
	}
	if _, err := service.CreateProvider(t.Context(), provider); err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	_, err := service.UpdateProviderSurfaceBinding(t.Context(), "derived-surface", testSurfaceBinding("provider-account-2", "derived-surface", "https://proxy.example.com/v1", providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED))
	if err == nil {
		t.Fatal("UpdateProviderSurfaceBinding() error = nil, want derived surface immutable error")
	}
	var validationErr *domainerror.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("UpdateProviderSurfaceBinding() error = %T, want ValidationError", err)
	}
}

func TestDeleteProviderSurfaceBindingRemovesSurfaceBindingFromProviderAggregate(t *testing.T) {
	service, store := newServiceForTest()
	provider := testProvider("provider-account-3")
	provider.Surfaces = []*providerv1.ProviderSurfaceBinding{
		testSurfaceBinding("provider-account-3", "surface-a", "https://api-a.example.com/v1", providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL),
		testSurfaceBinding("provider-account-3", "surface-b", "https://api-b.example.com/v1", providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL),
	}
	if _, err := service.CreateProvider(t.Context(), provider); err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if err := service.DeleteProviderSurfaceBinding(t.Context(), "surface-a"); err != nil {
		t.Fatalf("DeleteProviderSurfaceBinding() error = %v", err)
	}

	materialized, err := store.Get(t.Context(), "provider-account-3")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := len(materialized.GetSurfaces()), 1; got != want {
		t.Fatalf("surfaces len = %d, want %d", got, want)
	}
	if got, want := materialized.GetSurfaces()[0].GetSurfaceId(), "surface-b"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
}

func testProvider(providerID string) *providerv1.Provider {
	return &providerv1.Provider{
		ProviderId:  providerID,
		DisplayName: providerID,
		Surfaces:    []*providerv1.ProviderSurfaceBinding{},
	}
}

func testSurfaceBinding(_ string, surfaceID, baseURL string, origin providerv1.ProviderSurfaceOrigin) *providerv1.ProviderSurfaceBinding {
	return &providerv1.ProviderSurfaceBinding{
		SurfaceId: surfaceID,
		SourceRef: &providerv1.ProviderSurfaceSourceRef{
			Kind:      providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR,
			Id:        "openai",
			SurfaceId: surfaceID,
		},
		ProviderCredentialRef: &providerv1.ProviderCredentialRef{
			ProviderCredentialId: "credential-openai",
		},
		Runtime: &providerv1.ProviderSurfaceRuntime{
			DisplayName: surfaceID,
			Origin:      origin,
			Access: &providerv1.ProviderSurfaceRuntime_Api{
				Api: &providerv1.ProviderAPISurfaceRuntime{
					Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
					BaseUrl:  baseURL,
				},
			},
		},
	}
}
