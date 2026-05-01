package anthropicprovider

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providercontract "code-code.internal/agent-runtime-contract/provider"
	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers/protocolruntime"
)

const (
	surfaceID = "anthropic"
)

// Provider implements the Anthropic protocol surface using the official
// Anthropic Go SDK with custom base URL support.
type Provider struct{}

// NewProvider creates one Anthropic protocol provider implementation.
func NewProvider() *Provider {
	return &Provider{}
}

// Surface returns the stable provider surface metadata.
func (p *Provider) Surface() *providercontract.ProviderSurface {
	return &supportv1.Surface{
		SurfaceId:     surfaceID,
		ProductInfoId: "anthropic",
		Spec: &supportv1.Surface_Api{
			Api: &supportv1.ApiSurface{ApiEndpoints: []*supportv1.ApiEndpoint{{
				Protocol: apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC,
			}}},
		},
		AuthPolicyId:        "vendor.anthropic",
		EgressPolicyId:      "vendor.anthropic",
		ModelCatalogProbeId: "surface.anthropic",
	}
}

// NewRuntime creates one runtime bound to the supplied provider.
func (p *Provider) NewRuntime(provider *providerv1.Provider, credential *credentialcontract.ResolvedCredential) (providercontract.ProviderRuntime, error) {
	if provider == nil {
		return nil, fmt.Errorf("platformk8s/anthropic: provider is nil")
	}
	if provider.GetSurfaceId() != surfaceID {
		return nil, fmt.Errorf("platformk8s/anthropic: unsupported surface %q", provider.GetSurfaceId())
	}
	if credential == nil || credential.Kind != credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY || credential.GetApiKey() == nil {
		return nil, fmt.Errorf("platformk8s/anthropic: api key credential is required")
	}
	return &protocolruntime.BaseRuntime{
		Provider:   proto.Clone(provider).(*providerv1.Provider),
		Credential: proto.Clone(credential).(*credentialcontract.ResolvedCredential),
		Now:        time.Now,
	}, nil
}
