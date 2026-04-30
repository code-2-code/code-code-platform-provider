package providerconnect

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestNewVendorAPIKeyCandidatesApplyRequestedCatalogs(t *testing.T) {
	catalogs, err := newSurfaceCatalogSet([]*ProviderModelCatalogInput{{
		SurfaceID: "minimax-openai-compatible",
		Models:    []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
	}})
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	candidates, err := newVendorAPIKeyCandidates(&supportv1.Vendor{
		ProviderBindings: []*supportv1.VendorProviderBinding{{
			SurfaceTemplates: []*supportv1.ProviderSurfaceRuntimeTemplate{
				{
					SurfaceId: "minimax-openai-compatible",
					Runtime:   testAPISurfaceRuntime("MiniMax OpenAI Compatible", apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, "https://api.minimaxi.com/v1"),
					BootstrapCatalog: &providerv1.ProviderModelCatalog{
						Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "stale-static-model"}},
						Source: providerv1.CatalogSource_CATALOG_SOURCE_VENDOR_PRESET,
					},
				},
				{
					SurfaceId: "minimax-anthropic",
					Runtime:   testAPISurfaceRuntime("MiniMax Anthropic", apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC, "https://api.minimaxi.com/anthropic"),
					BootstrapCatalog: &providerv1.ProviderModelCatalog{
						Models: []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "claude-sonnet-4"}},
						Source: providerv1.CatalogSource_CATALOG_SOURCE_VENDOR_PRESET,
					},
				},
			},
		}},
	}, catalogs)
	if err != nil {
		t.Fatalf("newVendorAPIKeyCandidates() error = %v", err)
	}
	if got, want := len(candidates), 2; got != want {
		t.Fatalf("len(candidates) = %d, want %d", got, want)
	}
	runtime := candidates[0].Runtime()
	if got, want := runtime.GetCatalog().GetModels()[0].GetProviderModelId(), "gpt-4.1"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
	if got, want := candidates[1].Runtime().GetApi().GetBaseUrl(), "https://api.minimaxi.com/anthropic"; got != want {
		t.Fatalf("base_url = %q, want %q", got, want)
	}
}

func TestNewCustomAPIKeyCandidateBuildsManualSurface(t *testing.T) {
	catalogs, err := newSurfaceCatalogSet([]*ProviderModelCatalogInput{{
		SurfaceID: "openai-compatible",
		Models: []*providerv1.ProviderModelCatalogEntry{
			{ProviderModelId: "gpt-4.1"},
			{ProviderModelId: "gpt-4.1-mini"},
		},
	}})
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	candidate, err := newCustomAPIKeyCandidate("Custom OpenAI", &APIKeyConnectInput{
		BaseURL:  "https://example.com/v1",
		Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
	}, catalogs)
	if err != nil {
		t.Fatalf("newCustomAPIKeyCandidate() error = %v", err)
	}
	runtime := candidate.Runtime()
	if got, want := candidate.SurfaceID(), "openai-compatible"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got, want := runtime.GetOrigin(), providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_MANUAL; got != want {
		t.Fatalf("origin = %v, want %v", got, want)
	}
	if got, want := runtime.GetCatalog().GetModels()[1].GetProviderModelId(), "gpt-4.1-mini"; got != want {
		t.Fatalf("provider_model_id = %q, want %q", got, want)
	}
}

func TestNewCustomAPIKeyCandidateAllowsMissingCatalog(t *testing.T) {
	catalogs, err := newSurfaceCatalogSet(nil)
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	candidate, err := newCustomAPIKeyCandidate("Custom OpenAI", &APIKeyConnectInput{
		BaseURL:  "https://example.com/v1",
		Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
	}, catalogs)
	if err != nil {
		t.Fatalf("newCustomAPIKeyCandidate() error = %v", err)
	}
	if candidate.Runtime().GetCatalog() != nil {
		t.Fatal("catalog is non-nil, want nil")
	}
}

func TestNewVendorAPIKeyCandidatesAllowEmptyCatalogBeforeDiscovery(t *testing.T) {
	catalogs, err := newSurfaceCatalogSet(nil)
	if err != nil {
		t.Fatalf("newSurfaceCatalogSet() error = %v", err)
	}
	candidates, err := newVendorAPIKeyCandidates(&supportv1.Vendor{
		ProviderBindings: []*supportv1.VendorProviderBinding{{
			ProviderBinding: &supportv1.ProviderBinding{
				SurfaceId:             "mistral-openai-compatible",
				ModelCatalogProbeId:   "openai-compatible",
				QuotaProbeId:          "mistral-billing",
				EgressPolicyId:        "vendor.mistral",
				HeaderRewritePolicyId: "vendor.mistral",
			},
			SurfaceTemplates: []*supportv1.ProviderSurfaceRuntimeTemplate{{
				SurfaceId: "mistral-openai-compatible",
				Runtime:   testAPISurfaceRuntime("Mistral OpenAI Compatible", apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE, "https://api.mistral.ai/v1"),
				BootstrapCatalog: &providerv1.ProviderModelCatalog{
					Source: providerv1.CatalogSource_CATALOG_SOURCE_VENDOR_PRESET,
				},
			}},
		}},
	}, catalogs)
	if err != nil {
		t.Fatalf("newVendorAPIKeyCandidates() error = %v", err)
	}
	if got, want := len(candidates), 1; got != want {
		t.Fatalf("len(candidates) = %d, want %d", got, want)
	}
	runtime := candidates[0].Runtime()
	if got, want := runtime.GetModelCatalogProbeId(), "openai-compatible"; got != want {
		t.Fatalf("model_catalog_probe_id = %q, want %q", got, want)
	}
	if got, want := runtime.GetQuotaProbeId(), "mistral-billing"; got != want {
		t.Fatalf("quota_probe_id = %q, want %q", got, want)
	}
	if got, want := runtime.GetEgressRulesetId(), "vendor.mistral"; got != want {
		t.Fatalf("egress_ruleset_id = %q, want %q", got, want)
	}
}

func TestNewCLIOAuthCandidateUsesSupportProviderBinding(t *testing.T) {
	candidate, err := newCLIOAuthCandidate("Codex", "codex", &supportv1.CLI{
		CliId:    "codex",
		VendorId: "openai",
		Oauth: &supportv1.OAuthSupport{
			ProviderBinding: &supportv1.ProviderBinding{
				SurfaceId:             "openai-compatible",
				ModelCatalogProbeId:   "codex-oauth",
				QuotaProbeId:          "codex-quota",
				EgressPolicyId:        "cli.openai-oauth",
				HeaderRewritePolicyId: "cli.openai-oauth",
			},
		},
	})
	if err != nil {
		t.Fatalf("newCLIOAuthCandidate() error = %v", err)
	}
	if got, want := candidate.SurfaceID(), "openai-compatible"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	runtime := candidate.Runtime()
	if got, want := runtime.GetCli().GetCliId(), "codex"; got != want {
		t.Fatalf("cli_id = %q, want %q", got, want)
	}
	if got, want := runtime.GetModelCatalogProbeId(), "codex-oauth"; got != want {
		t.Fatalf("model_catalog_probe_id = %q, want %q", got, want)
	}
	if got, want := runtime.GetQuotaProbeId(), "codex-quota"; got != want {
		t.Fatalf("quota_probe_id = %q, want %q", got, want)
	}
	if got, want := runtime.GetEgressRulesetId(), "cli.openai-oauth"; got != want {
		t.Fatalf("egress_ruleset_id = %q, want %q", got, want)
	}
}
