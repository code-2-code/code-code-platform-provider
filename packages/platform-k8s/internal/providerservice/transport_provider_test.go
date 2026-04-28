package providerservice

import (
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderViewToServicePreservesCredentialFields(t *testing.T) {
	view := &managementv1.ProviderView{
		ProviderId:           "provider-1",
		DisplayName:          "Provider",
		VendorId:             "mistral",
		ProviderCredentialId: "credential-1",
		ModelCatalog:         &providerv1.ProviderModelCatalog{},
		Surfaces: []*managementv1.ProviderSurfaceBindingView{{
			SurfaceId:            "surface-1",
			ProviderCredentialId: "credential-1",
			VendorId:             "mistral",
		}},
	}

	out := providerViewToService(view)

	if got, want := out.GetProviderCredentialId(), "credential-1"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
	if out.GetModelCatalog() == nil {
		t.Fatal("model_catalog = nil, want value")
	}
	if got, want := out.GetSurfaces()[0].GetProviderCredentialId(), "credential-1"; got != want {
		t.Fatalf("surface provider_credential_id = %q, want %q", got, want)
	}
}

func TestProviderConnectProviderToTransportPreservesSurfaceCredentialFields(t *testing.T) {
	source := &managementv1.ProviderView{
		ProviderId:           "provider-1",
		DisplayName:          "Provider",
		VendorId:             "mistral",
		ProviderCredentialId: "credential-1",
		Surfaces: []*managementv1.ProviderSurfaceBindingView{{
			SurfaceId:            "surface-1",
			ProviderCredentialId: "credential-1",
			VendorId:             "mistral",
			ProviderId:           "provider-1",
			ProviderDisplayName:  "Provider",
		}},
	}
	view := providerConnectProviderFromTransport(source)

	out := providerConnectProviderToTransport(view)

	if got, want := out.GetProviderCredentialId(), "credential-1"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
	surface := out.GetSurfaces()[0]
	if got, want := surface.GetProviderCredentialId(), "credential-1"; got != want {
		t.Fatalf("surface provider_credential_id = %q, want %q", got, want)
	}
	if got, want := surface.GetVendorId(), "mistral"; got != want {
		t.Fatalf("surface vendor_id = %q, want %q", got, want)
	}
}
