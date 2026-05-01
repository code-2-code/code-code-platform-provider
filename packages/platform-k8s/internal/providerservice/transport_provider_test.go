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
		Models:               []*providerv1.ProviderModel{{ProviderModelId: "model-1"}},
		Endpoints: []*providerv1.ProviderEndpoint{{
			Type:  providerv1.ProviderEndpointType_PROVIDER_ENDPOINT_TYPE_CLI,
			Shape: &providerv1.ProviderEndpoint_Cli{Cli: &providerv1.ProviderCliEndpoint{CliId: "codex"}},
		}},
		Status: &managementv1.ProviderStatus{Phase: providerservicev1.ProviderPhase_PROVIDER_PHASE_READY},
	}

	out := providerViewToService(view)

	if got, want := out.GetProviderCredentialId(), "credential-1"; got != want {
		t.Fatalf("provider_credential_id = %q, want %q", got, want)
	}
	if got, want := out.GetModels()[0].GetProviderModelId(), "model-1"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
	if got, want := out.GetEndpoints()[0].GetCli().GetCliId(), "codex"; got != want {
		t.Fatalf("endpoint cli_id = %q, want %q", got, want)
	}
	if got, want := out.GetStatus().GetPhase(), providerservicev1.ProviderPhase_PROVIDER_PHASE_READY; got != want {
		t.Fatalf("status.phase = %v, want %v", got, want)
	}
}
