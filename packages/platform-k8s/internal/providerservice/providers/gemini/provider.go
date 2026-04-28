package geminiprovider

import (
	"fmt"
	"time"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providercontract "code-code.internal/agent-runtime-contract/provider"
	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
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
	return &providerv1.ProviderSurface{
		SurfaceId:                surfaceID,
		DisplayName:              "Gemini",
		SupportedCredentialKinds: []credentialv1.CredentialKind{credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY},
		Kind:                     providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API,
		Api: &providerv1.ProviderSurfaceAPISpec{
			SupportedProtocols: []apiprotocolv1.Protocol{apiprotocolv1.Protocol_PROTOCOL_GEMINI},
		},
		Capabilities: &providerv1.ProviderCapabilities{
			SupportsModelOverride: false,
		},
	}
}

func (p *Provider) NewRuntime(
	surface *providercontract.ProviderSurfaceBinding,
	credential *credentialcontract.ResolvedCredential,
) (providercontract.ProviderRuntime, error) {
	if surface == nil {
		return nil, fmt.Errorf("platformk8s/gemini: provider surface is nil")
	}
	runtime := surface.GetRuntime()
	if runtime == nil {
		return nil, fmt.Errorf("platformk8s/gemini: provider surface runtime is required")
	}
	api := runtime.GetApi()
	if api.GetProtocol() != apiprotocolv1.Protocol_PROTOCOL_GEMINI {
		return nil, fmt.Errorf("platformk8s/gemini: unsupported protocol %s", api.GetProtocol().String())
	}
	if api.GetBaseUrl() == "" {
		return nil, fmt.Errorf("platformk8s/gemini: base_url is required")
	}
	if credential == nil || credential.Kind != credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY || credential.GetApiKey() == nil {
		return nil, fmt.Errorf("platformk8s/gemini: api key credential is required")
	}
	return &protocolruntime.BaseRuntime{
		Surface:    proto.Clone(surface).(*providerv1.ProviderSurfaceBinding),
		Credential: proto.Clone(credential).(*credentialcontract.ResolvedCredential),
		Now:        time.Now,
	}, nil
}
