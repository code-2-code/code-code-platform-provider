package providers

import (
	"context"
	"fmt"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	"code-code.internal/go-contract/domainerror"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

func TestListGroupsConsistentProvider(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testProviderSurfaceBindingView("openai-surface", "Provider A", "projection-a", "credential-a", "vendor-a", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
		testProviderSurfaceBindingView("anthropic-surface", "Provider A", "projection-a", "credential-a", "vendor-a", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
	})

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	if got, want := items[0].GetProviderId(), "projection-a"; got != want {
		t.Fatalf("provider_id = %q, want %q", got, want)
	}
	if got, want := items[0].GetProviderCredentialId(), "credential-a"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
	if got, want := len(items[0].GetSurfaces()), 2; got != want {
		t.Fatalf("len(surfaces) = %d, want %d", got, want)
	}
}

func TestListProjectsConflictingCredentialIDs(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testProviderSurfaceBindingView("surface-a", "Provider A", "projection-a", "credential-a", "vendor-a", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
		testProviderSurfaceBindingView("surface-b", "Provider A", "projection-a", "credential-b", "vendor-a", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
	})

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := items[0].GetProviderCredentialId(), "credential-a"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
}

func TestListProjectsMixedSurfaceKinds(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testProviderSurfaceBindingView("surface-a", "Provider A", "projection-a", "credential-a", "vendor-a", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
		testCLIProviderSurfaceBindingView("surface-b", "Provider A", "projection-a", "credential-a", "vendor-a", "codex"),
	})

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(items[0].GetSurfaces()), 2; got != want {
		t.Fatalf("len(surfaces) = %d, want %d", got, want)
	}
}

func TestListProjectsConflictingCliIDs(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testCLIProviderSurfaceBindingView("surface-a", "Provider A", "projection-a", "credential-a", "vendor-a", "codex"),
		testCLIProviderSurfaceBindingView("surface-b", "Provider A", "projection-a", "credential-a", "vendor-a", "qwen-cli"),
	})

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := items[0].GetProviderCredentialId(), "credential-a"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
}

func TestListProjectsVendorAPIKeyIconURL(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testProviderSurfaceBindingView("surface-a", "Provider A", "projection-a", "credential-a", "openai", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
	})
	service.vendors = stubVendorReferenceService{
		items: []*managementv1.VendorView{{
			VendorId: "openai",
			IconUrl:  "https://openai.com/favicon.ico",
		}},
	}

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := items[0].GetIconUrl(), "https://openai.com/favicon.ico"; got != want {
		t.Fatalf("icon_url = %q, want %q", got, want)
	}
}

func TestListProjectsCLIOAuthIconURL(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testCLIProviderSurfaceBindingView("surface-a", "Provider A", "projection-a", "credential-a", "openai", "codex"),
	})
	service.vendors = stubVendorReferenceService{
		items: []*managementv1.VendorView{{
			VendorId: "openai",
			IconUrl:  "https://openai.com/favicon.ico",
		}},
	}
	service.cliDefs = stubCLIDefinitionReferenceService{
		items: []*managementv1.CLIDefinitionView{{
			CliId:   "codex",
			IconUrl: "https://codex.example/icon.png",
		}},
	}

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := items[0].GetIconUrl(), "https://codex.example/icon.png"; got != want {
		t.Fatalf("icon_url = %q, want %q", got, want)
	}
}

func TestListSkipsInstanceWithoutProviderID(t *testing.T) {
	service := newListTestService([]*managementv1.ProviderSurfaceBindingView{
		testProviderSurfaceBindingView("surface-a", "Provider A", "projection-a", "credential-a", "vendor-a", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
		testProviderSurfaceBindingView("surface-b", "Provider B", "", "credential-b", "vendor-b", providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API),
	})

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	if got, want := items[0].GetProviderId(), "projection-a"; got != want {
		t.Fatalf("provider_id = %q, want %q", got, want)
	}
}

func newListTestService(instances []*managementv1.ProviderSurfaceBindingView) *Service {
	return &Service{repository: testProviderStore{providers: providersFromSurfaceViews(instances)}}
}

type testProviderStore struct {
	providers []*providerv1.Provider
}

func (s testProviderStore) List(context.Context) ([]*providerv1.Provider, error) {
	items := make([]*providerv1.Provider, 0, len(s.providers))
	for _, provider := range s.providers {
		items = append(items, proto.Clone(provider).(*providerv1.Provider))
	}
	return items, nil
}

func (s testProviderStore) Get(_ context.Context, providerID string) (*providerv1.Provider, error) {
	for _, provider := range s.providers {
		if provider.GetProviderId() == providerID {
			return proto.Clone(provider).(*providerv1.Provider), nil
		}
	}
	return nil, domainerror.NewNotFound("provider %q not found", providerID)
}

func (s testProviderStore) Upsert(context.Context, *providerv1.Provider) (*providerv1.Provider, error) {
	return nil, fmt.Errorf("unexpected Upsert")
}

func (s testProviderStore) Update(context.Context, string, func(*providerv1.Provider) error) (*providerv1.Provider, error) {
	return nil, fmt.Errorf("unexpected Update")
}

func (s testProviderStore) Delete(context.Context, string) error {
	return fmt.Errorf("unexpected Delete")
}

func providersFromSurfaceViews(instances []*managementv1.ProviderSurfaceBindingView) []*providerv1.Provider {
	providers := map[string]*providerv1.Provider{}
	for _, instance := range instances {
		providerID := instance.GetProviderId()
		if providerID == "" {
			continue
		}
		provider := providers[providerID]
		if provider == nil {
			provider = &providerv1.Provider{
				ProviderId:  providerID,
				DisplayName: instance.GetProviderDisplayName(),
			}
			providers[providerID] = provider
		}
		surface := &providerv1.ProviderSurfaceBinding{
			SurfaceId: instance.GetSurfaceId(),
			Runtime:   proto.Clone(instance.GetRuntime()).(*providerv1.ProviderSurfaceRuntime),
			SourceRef: providerSurfaceBindingSourceRefForTest(instance.GetVendorId(), instance.GetSurfaceId()),
			ProviderCredentialRef: &providerv1.ProviderCredentialRef{
				ProviderCredentialId: instance.GetProviderCredentialId(),
			},
		}
		provider.Surfaces = append(provider.Surfaces, surface)
	}
	items := make([]*providerv1.Provider, 0, len(providers))
	for _, provider := range providers {
		items = append(items, provider)
	}
	return items
}

func providerSurfaceBindingSourceRefForTest(vendorID, surfaceID string) *providerv1.ProviderSurfaceSourceRef {
	if vendorID == "" {
		return nil
	}
	return &providerv1.ProviderSurfaceSourceRef{
		Kind:      providerv1.ProviderSurfaceSourceKind_PROVIDER_SURFACE_SOURCE_KIND_VENDOR,
		Id:        vendorID,
		SurfaceId: surfaceID,
	}
}

type stubProviderSurfaceBindingService struct {
	instances []*managementv1.ProviderSurfaceBindingView
}

func (s stubProviderSurfaceBindingService) ListInstances(context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	return s.instances, nil
}

func (stubProviderSurfaceBindingService) UpdateProviderDisplayName(context.Context, string, string) error {
	return nil
}

func (stubProviderSurfaceBindingService) DeleteInstance(context.Context, string) error {
	return nil
}

type stubVendorReferenceService struct {
	items []*managementv1.VendorView
}

func (s stubVendorReferenceService) List(context.Context) ([]*managementv1.VendorView, error) {
	return s.items, nil
}

type stubCLIDefinitionReferenceService struct {
	items []*managementv1.CLIDefinitionView
}

func (s stubCLIDefinitionReferenceService) List(context.Context) ([]*managementv1.CLIDefinitionView, error) {
	return s.items, nil
}

func testProviderSurfaceBindingView(
	instanceID, providerDisplayName, providerID, credentialID, vendorID string,
	kind providerv1.ProviderSurfaceKind,
) *managementv1.ProviderSurfaceBindingView {
	return &managementv1.ProviderSurfaceBindingView{
		SurfaceId:            instanceID,
		DisplayName:          instanceID,
		ProviderId:           providerID,
		ProviderDisplayName:  providerDisplayName,
		ProviderCredentialId: credentialID,
		VendorId:             vendorID,
		Runtime:              testSurfaceRuntimeForKind(instanceID, kind),
	}
}

func testCLIProviderSurfaceBindingView(
	instanceID, providerDisplayName, providerID, credentialID, vendorID, cliID string,
) *managementv1.ProviderSurfaceBindingView {
	view := testProviderSurfaceBindingView(
		instanceID,
		providerDisplayName,
		providerID,
		credentialID,
		vendorID,
		providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI,
	)
	view.Runtime = &providerv1.ProviderSurfaceRuntime{
		DisplayName: instanceID,
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
		Access: &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: cliID},
		},
	}
	return view
}

func testSurfaceRuntimeForKind(
	displayName string,
	kind providerv1.ProviderSurfaceKind,
) *providerv1.ProviderSurfaceRuntime {
	runtime := &providerv1.ProviderSurfaceRuntime{
		DisplayName: displayName,
		Origin:      providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED,
	}
	if kind == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI {
		runtime.Access = &providerv1.ProviderSurfaceRuntime_Cli{
			Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: "codex"},
		}
		return runtime
	}
	runtime.Access = &providerv1.ProviderSurfaceRuntime_Api{
		Api: &providerv1.ProviderAPISurfaceRuntime{
			Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
			BaseUrl:  "https://api.example.com/v1",
		},
	}
	return runtime
}
