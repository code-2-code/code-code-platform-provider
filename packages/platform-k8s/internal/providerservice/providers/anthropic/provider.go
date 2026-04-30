package anthropicprovider

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providercontract "code-code.internal/agent-runtime-contract/provider"
	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
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
	return &providerv1.ProviderSurface{
		SurfaceId:                surfaceID,
		DisplayName:              "Anthropic",
		SupportedCredentialKinds: []credentialv1.CredentialKind{credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY},
		Kind:                     providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
		Spec: &providerv1.ProviderSurface_Api{
			Api: &providerv1.ProviderSurfaceAPISpec{
				SupportedProtocols: []apiprotocolv1.Protocol{apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC},
			},
		},
		Capabilities: &providerv1.ProviderCapabilities{
			SupportsModelOverride: true,
		},
	}
}

// NewRuntime creates one runtime bound to the supplied provider.
func (p *Provider) NewRuntime(provider *providerv1.Provider, credential *credentialcontract.ResolvedCredential) (providercontract.ProviderRuntime, error) {
	if provider == nil {
		return nil, fmt.Errorf("platformk8s/anthropic: provider is nil")
	}
	runtime := provider.GetRuntime()
	if runtime == nil {
		return nil, fmt.Errorf("platformk8s/anthropic: provider surface runtime is required")
	}
	api := runtime.GetApi()
	if api.GetProtocol() != apiprotocolv1.Protocol_PROTOCOL_ANTHROPIC {
		return nil, fmt.Errorf("platformk8s/anthropic: unsupported protocol %s", api.GetProtocol().String())
	}
	if api.GetBaseUrl() == "" {
		return nil, fmt.Errorf("platformk8s/anthropic: base_url is required")
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
