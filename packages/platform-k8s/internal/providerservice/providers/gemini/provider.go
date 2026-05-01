package geminiprovider

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

const surfaceID = "gemini"

// Provider implements the Gemini native API surface.
type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Surface() *providercontract.ProviderSurface {
	return &supportv1.Surface{
		SurfaceId:     surfaceID,
		ProductInfoId: "google-ai-studio",
		Spec: &supportv1.Surface_Api{
			Api: &supportv1.ApiSurface{ApiEndpoints: []*supportv1.ApiEndpoint{{
				Protocol: apiprotocolv1.Protocol_PROTOCOL_GEMINI,
			}}},
		},
		AuthPolicyId:        "vendor.google",
		EgressPolicyId:      "vendor.google",
		ModelCatalogProbeId: "surface.gemini",
		QuotaProbeId:        "google-ai-studio",
	}
}

func (p *Provider) NewRuntime(
	provider *providerv1.Provider,
	credential *credentialcontract.ResolvedCredential,
) (providercontract.ProviderRuntime, error) {
	if provider == nil {
		return nil, fmt.Errorf("platformk8s/gemini: provider is nil")
	}
	if provider.GetSurfaceId() != surfaceID {
		return nil, fmt.Errorf("platformk8s/gemini: unsupported surface %q", provider.GetSurfaceId())
	}
	if credential == nil || credential.Kind != credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY || credential.GetApiKey() == nil {
		return nil, fmt.Errorf("platformk8s/gemini: api key credential is required")
	}
	return &protocolruntime.BaseRuntime{
		Provider:   proto.Clone(provider).(*providerv1.Provider),
		Credential: proto.Clone(credential).(*credentialcontract.ResolvedCredential),
		Now:        time.Now,
	}, nil
}
