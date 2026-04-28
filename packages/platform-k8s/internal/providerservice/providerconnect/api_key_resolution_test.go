package providerconnect

import (
	"context"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	vendordefinitionv1 "code-code.internal/go-contract/vendor_definition/v1"
)

func TestProviderConnectAPIKeyResolutionRuntimeResolveCustom(t *testing.T) {
	runtime := newProviderConnectAPIKeyResolutionRuntime(
		providerConnectSupport{},
		newProviderConnectQueries(
			nil,
			nil,
			definitionReaderStub{items: map[string]*providerv1.ProviderSurface{
				"openai-compatible": testProviderSurface(
					"openai-compatible",
					providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
					[]credentialv1.CredentialKind{credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY},
					apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
				),
			}},
		),
	)
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod:   AddMethodAPIKey,
		DisplayName: "Custom OpenAI",
		APIKey: &APIKeyConnectInput{
			APIKey:   "sk-test",
			BaseURL:  "https://example.com/v1",
			Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
			SurfaceModelCatalogs: []*ProviderSurfaceBindingModelCatalogInput{{
				SurfaceID: "openai-compatible",
				Models:    []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}

	resolved, err := runtime.Resolve(context.Background(), command)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.target == nil {
		t.Fatal("target = nil, want non-nil")
	}
	if got, want := resolved.target.DisplayName, "Custom OpenAI"; got != want {
		t.Fatalf("display_name = %q, want %q", got, want)
	}
	if got, want := resolved.target.RuntimeTemplate.GetCatalog().GetModels()[0].GetProviderModelId(), "gpt-4.1"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}

func TestProviderConnectAPIKeyResolutionRuntimeResolveVendor(t *testing.T) {
	runtime := newProviderConnectAPIKeyResolutionRuntime(
		newProviderConnectSupport(
			vendorSupportReaderStub{items: map[string]*supportv1.Vendor{
				"openai": {
					Vendor: &vendordefinitionv1.Vendor{
						VendorId:    "openai",
						DisplayName: "OpenAI",
					},
					ProviderBindings: []*supportv1.VendorProviderBinding{{
						SurfaceTemplates: []*supportv1.ProviderSurfaceRuntimeTemplate{{
							SurfaceId: "openai-compatible",
							Runtime:   testAPISurfaceRuntime("OpenAI Compatible", apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, "https://api.openai.com/v1"),
							BootstrapCatalog: &providerv1.ProviderModelCatalog{
								Source: providerv1.CatalogSource_CATALOG_SOURCE_VENDOR_PRESET,
								Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
							},
						}},
					}},
				},
			}},
			nil,
		),
		newProviderConnectQueries(
			nil,
			nil,
			definitionReaderStub{items: map[string]*providerv1.ProviderSurface{
				"openai-compatible": testProviderSurface(
					"openai-compatible",
					providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
					[]credentialv1.CredentialKind{credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY},
					apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
				),
			}},
		),
	)
	command, err := NewConnectCommand(ConnectCommandInput{
		AddMethod: AddMethodAPIKey,
		VendorID:  "openai",
		APIKey: &APIKeyConnectInput{
			APIKey: "sk-openai",
		},
	})
	if err != nil {
		t.Fatalf("NewConnectCommand() error = %v", err)
	}

	resolved, err := runtime.Resolve(context.Background(), command)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.plan == nil {
		t.Fatal("plan = nil, want non-nil")
	}
	if got, want := resolved.plan.DisplayName, "OpenAI"; got != want {
		t.Fatalf("display_name = %q, want %q", got, want)
	}
	if got, want := resolved.plan.VendorID, "openai"; got != want {
		t.Fatalf("vendor_id = %q, want %q", got, want)
	}
	if got, want := len(resolved.plan.Targets), 1; got != want {
		t.Fatalf("len(targets) = %d, want %d", got, want)
	}
	if got, want := resolved.plan.Targets[0].TargetCredentialID, resolved.plan.TargetCredentialID; got != want {
		t.Fatalf("target credential id = %q, want %q", got, want)
	}
	if got, want := resolved.plan.Targets[0].TargetProviderID, resolved.plan.TargetProviderID; got != want {
		t.Fatalf("target provider id = %q, want %q", got, want)
	}
}

type vendorSupportReaderStub struct {
	items map[string]*supportv1.Vendor
}

func (s vendorSupportReaderStub) GetForConnect(
	_ context.Context,
	vendorID string,
) (*supportv1.Vendor, error) {
	return s.items[vendorID], nil
}
