package providerconnect

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestConnectSurfaceMetadataValidateCandidateRejectsUnsupportedCredentialKind(t *testing.T) {
	catalogs, err := newSurfaceCatalogSet([]*ProviderSurfaceBindingModelCatalogInput{{
		SurfaceID: "openai-compatible",
		Models:    []*providerv1.ProviderModelCatalogEntry{{ProviderModelId: "gpt-4.1"}},
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
	definition, err := newConnectSurfaceMetadata(testProviderSurface(
		"openai-compatible",
		providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
		[]credentialv1.CredentialKind{credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH},
		apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
	))
	if err != nil {
		t.Fatalf("newConnectSurfaceMetadata() error = %v", err)
	}

	err = definition.ValidateCandidate(candidate, credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY)
	if err == nil {
		t.Fatal("ValidateCandidate() error = nil, want validation error")
	}
}
