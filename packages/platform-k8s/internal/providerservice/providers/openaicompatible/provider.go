package openaicompatible

import (
	"fmt"
	"time"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providercontract "code-code.internal/agent-runtime-contract/provider"
	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers/protocolruntime"
	"google.golang.org/protobuf/proto"
)

const (
	surfaceID = "openai-compatible"
)

// Provider implements the OpenAI-compatible protocol surface using the official
// OpenAI Go SDK with custom base URL support.
type Provider struct{}

// NewProvider creates one OpenAI-compatible provider implementation.
func NewProvider() *Provider {
	return &Provider{}
}

// Surface returns the stable provider surface metadata.
func (p *Provider) Surface() *providercontract.ProviderSurface {
	return &supportv1.Surface{
		SurfaceId:     surfaceID,
		ProductInfoId: "openai",
		Spec: &supportv1.Surface_Api{
			Api: &supportv1.ApiSurface{ApiEndpoints: []*supportv1.ApiEndpoint{
				{Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE},
				{Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_RESPONSES},
			}},
		},
		AuthPolicyId:        "vendor.openai",
		EgressPolicyId:      "vendor.openai",
		ModelCatalogProbeId: "surface.openai-compatible",
	}
}

// NewRuntime creates one runtime bound to the supplied provider.
func (p *Provider) NewRuntime(provider *providerv1.Provider, credential *credentialcontract.ResolvedCredential) (providercontract.ProviderRuntime, error) {
	if provider == nil {
		return nil, fmt.Errorf("platformk8s/openaicompatible: provider is nil")
	}
	if provider.GetSurfaceId() != surfaceID {
		return nil, fmt.Errorf("platformk8s/openaicompatible: unsupported surface %q", provider.GetSurfaceId())
	}
	if credential == nil || credential.Kind != credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY || credential.GetApiKey() == nil {
		return nil, fmt.Errorf("platformk8s/openaicompatible: api key credential is required")
	}
	return &protocolruntime.BaseRuntime{
		Provider:   proto.Clone(provider).(*providerv1.Provider),
		Credential: proto.Clone(credential).(*credentialcontract.ResolvedCredential),
		Now:        time.Now,
	}, nil
}
