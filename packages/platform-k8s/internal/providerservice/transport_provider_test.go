package providerservice

import (
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderViewToServicePreservesCredentialFields(t *testing.T) {
	view := &managementv1.ProviderView{
		ProviderId:           "provider-1",
		DisplayName:          "Provider",
		ProviderCredentialId: "credential-1",
		ModelCatalog:         &providerv1.ProviderModelCatalog{},
		ProductInfoId:        "product-1",
		Status:               &managementv1.ProviderStatus{Phase: providerservicev1.ProviderPhase_PROVIDER_PHASE_READY},
	}

	out := providerViewToService(view)

	if got, want := out.GetProviderCredentialId(), "credential-1"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
	if out.GetModelCatalog() == nil {
		t.Fatal("model_catalog = nil, want value")
	}
	if got, want := out.GetProductInfoId(), "product-1"; got != want {
		t.Fatalf("product_info.id = %q, want %q", got, want)
	}
	if got, want := out.GetStatus().GetPhase(), providerservicev1.ProviderPhase_PROVIDER_PHASE_READY; got != want {
		t.Fatalf("status.phase = %v, want %v", got, want)
	}
}
