package providercatalogs

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
)

func TestCatalogProbeLeaseKeyUsesSourceConcurrencyKey(t *testing.T) {
	operation := &modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation{
		Path:         "models",
		ResponseKind: modelcatalogdiscoveryv1.ModelCatalogDiscoveryResponseKind_MODEL_CATALOG_DISCOVERY_RESPONSE_KIND_OPENAI_MODELS,
	}
	requestA := CatalogProbeRequest{
		Protocol:                 apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		BaseURL:                  "https://api.example.test/v1",
		ProviderSurfaceBindingID: "surface-a",
		ConcurrencyKey:           "probe.example",
	}
	requestB := requestA
	requestB.ProviderSurfaceBindingID = "surface-b"

	keyA := catalogProbeSingleflightKey(requestA, operation)
	keyB := catalogProbeSingleflightKey(requestB, operation)
	if keyA == keyB {
		t.Fatalf("singleflight keys should remain request-specific")
	}
	if got, want := catalogProbeLeaseKey(requestA, keyA), "probe.example"; got != want {
		t.Fatalf("lease key = %q, want %q", got, want)
	}
	if got, want := catalogProbeLeaseKey(CatalogProbeRequest{}, keyA), keyA; got != want {
		t.Fatalf("fallback lease key = %q, want %q", got, want)
	}
}
