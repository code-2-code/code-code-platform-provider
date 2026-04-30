package providerservice

import (
	"context"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	providers "code-code.internal/platform-k8s/internal/providerservice/providers"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
)

func TestListProvidersReturnsProviders(t *testing.T) {
	server := newProviderReadTestServer(t)

	response, err := server.ListProviders(context.Background(), &providerservicev1.ListProvidersRequest{})
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if got, want := len(response.GetItems()), 1; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	if got, want := response.GetItems()[0].GetProviderId(), "sample-openai-compatible"; got != want {
		t.Fatalf("provider_id = %q, want %q", got, want)
	}
}

func TestListProviderSurfacesReturnsBuiltins(t *testing.T) {
	server := newProviderReadTestServer(t)

	response, err := server.ListProviderSurfaces(context.Background(), &providerservicev1.ListProviderSurfacesRequest{})
	if err != nil {
		t.Fatalf("ListProviderSurfaces() error = %v", err)
	}
	if !hasProviderSurface(response.GetItems(), "openai-compatible") {
		t.Fatalf("ListProviderSurfaces() missing openai-compatible: %v", response.GetItems())
	}
}

func newProviderReadTestServer(t *testing.T) *Server {
	t.Helper()
	surfaces, err := providersurfaces.NewService()
	if err != nil {
		t.Fatalf("NewService(providersurfaces) error = %v", err)
	}
	return &Server{surfaceMetadata: surfaces, providers: providerReadService{items: []*managementv1.ProviderView{providerReadTestView()}}}
}

func providerReadTestView() *managementv1.ProviderView {
	return &managementv1.ProviderView{
		ProviderId:  "sample-openai-compatible",
		DisplayName: "Sample OpenAI",
	}
}

func hasProviderSurface(items []*providerv1.ProviderSurface, surfaceID string) bool {
	for _, item := range items {
		if item.GetSurfaceId() == surfaceID {
			return true
		}
	}
	return false
}

type providerReadService struct {
	items []*managementv1.ProviderView
}

func (s providerReadService) List(context.Context) ([]*managementv1.ProviderView, error) {
	return s.items, nil
}

func (s providerReadService) Get(context.Context, string) (*managementv1.ProviderView, error) {
	return nil, nil
}

func (s providerReadService) Update(context.Context, string, providers.UpdateProviderCommand) (*managementv1.ProviderView, error) {
	return nil, nil
}

func (s providerReadService) Delete(context.Context, string) error {
	return nil
}
