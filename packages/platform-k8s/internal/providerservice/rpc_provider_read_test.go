package providerservice

import (
	"context"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	providers "code-code.internal/platform-k8s/internal/providerservice/providers"
	clisupport "code-code.internal/platform-k8s/internal/supportservice/clidefinitions/support"
	"code-code.internal/platform-k8s/internal/supportservice/providersurfaces"
	vendorsupport "code-code.internal/platform-k8s/internal/supportservice/vendors/support"
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
	cliSupport, err := clisupport.NewManagementService()
	if err != nil {
		t.Fatalf("NewManagementService() error = %v", err)
	}
	vendorSupport, err := vendorsupport.NewManagementService()
	if err != nil {
		t.Fatalf("NewManagementService(vendors) error = %v", err)
	}
	surfaces, err := providersurfaces.NewService(cliSupport, vendorSupport)
	if err != nil {
		t.Fatalf("NewService(providersurfaces) error = %v", err)
	}
	return &Server{surfaceMetadata: surfaces, providers: providerReadService{items: []*managementv1.ProviderView{providerReadTestView()}}}
}

func providerReadTestView() *managementv1.ProviderView {
	return &managementv1.ProviderView{
		ProviderId:  "sample-openai-compatible",
		DisplayName: "Sample OpenAI",
		VendorId:    "openai",
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

func (s providerReadService) UpdateAPIKeyAuthentication(context.Context, string, providers.UpdateAPIKeyAuthenticationCommand) (*managementv1.UpdateProviderAuthenticationResponse, error) {
	return nil, nil
}

func (s providerReadService) UpdateObservabilityAuthentication(context.Context, string, providers.UpdateObservabilityAuthenticationCommand) (*managementv1.ProviderView, error) {
	return nil, nil
}

func (s providerReadService) Delete(context.Context, string) error {
	return nil
}
