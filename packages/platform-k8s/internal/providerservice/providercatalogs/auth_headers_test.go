package providercatalogs

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"code-code.internal/platform-k8s/internal/platform/modelcatalogdiscovery"
	"google.golang.org/grpc"
)

func TestEgressAuthCatalogProbeHeaderResolverGeneratesBearerHeader(t *testing.T) {
	t.Parallel()

	auth := &fakeCatalogProbeAuthClient{
		policy: &authv1.GetEgressAuthPolicyResponse{
			PolicyId:           "protocol.openai-compatible.api-key",
			HeaderValuePrefix:  "Bearer",
			RequestHeaderNames: []string{"authorization"},
			RequestReplacementRules: []*authv1.EgressSimpleReplacementRule{{
				Mode:              "bearer",
				HeaderName:        "authorization",
				MaterialKey:       "api_key",
				HeaderValuePrefix: "Bearer",
			}},
		},
		headers: &authv1.ResolveEgressRequestHeadersResponse{
			Headers: []*authv1.EgressHeaderMutation{{
				Name:  "authorization",
				Value: "Bearer token-1",
			}},
		},
	}
	resolver := NewEgressAuthCatalogProbeHeaderResolver(auth)

	headers, err := resolver.ResolveCatalogProbeHeaders(context.Background(), CatalogProbeHeaderRequest{
		CredentialID: "credential-mistral",
		Protocol:     apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		BaseURL:      "https://api.mistral.ai/v1",
		Operation:    modelcatalogdiscovery.DefaultAPIKeyDiscoveryOperation(apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE),
	})
	if err != nil {
		t.Fatalf("ResolveCatalogProbeHeaders() error = %v", err)
	}
	if got, want := auth.policyRequest.GetCredentialKind(), credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY; got != want {
		t.Fatalf("credential_kind = %s, want %s", got, want)
	}
	if got, want := auth.policyRequest.GetProtocol(), apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE; got != want {
		t.Fatalf("protocol = %s, want %s", got, want)
	}
	if got, want := auth.resolveRequest.GetCredentialId(), "credential-mistral"; got != want {
		t.Fatalf("credential_id = %q, want %q", got, want)
	}
	if got, want := auth.resolveRequest.GetTargetHost(), "api.mistral.ai"; got != want {
		t.Fatalf("target_host = %q, want %q", got, want)
	}
	if got, want := auth.resolveRequest.GetTargetPath(), "/v1/models"; got != want {
		t.Fatalf("target_path = %q, want %q", got, want)
	}
	if got, want := headers.Get("Authorization"), "Bearer token-1"; got != want {
		t.Fatalf("authorization = %q, want %q", got, want)
	}
}

func TestEgressAuthCatalogProbeHeaderResolverRejectsMissingCredential(t *testing.T) {
	t.Parallel()

	resolver := NewEgressAuthCatalogProbeHeaderResolver(&fakeCatalogProbeAuthClient{})
	_, err := resolver.ResolveCatalogProbeHeaders(context.Background(), CatalogProbeHeaderRequest{
		Protocol:  apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
		BaseURL:   "https://api.mistral.ai/v1",
		Operation: modelcatalogdiscovery.DefaultAPIKeyDiscoveryOperation(apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE),
	})
	if err == nil || !strings.Contains(err.Error(), "provider credential is required") {
		t.Fatalf("error = %v, want provider credential validation", err)
	}
}

type fakeCatalogProbeAuthClient struct {
	policyRequest  *authv1.GetEgressAuthPolicyRequest
	resolveRequest *authv1.ResolveEgressRequestHeadersRequest
	policy         *authv1.GetEgressAuthPolicyResponse
	headers        *authv1.ResolveEgressRequestHeadersResponse
	err            error
}

func (c *fakeCatalogProbeAuthClient) GetEgressAuthPolicy(_ context.Context, request *authv1.GetEgressAuthPolicyRequest, _ ...grpc.CallOption) (*authv1.GetEgressAuthPolicyResponse, error) {
	c.policyRequest = request
	return c.policy, c.err
}

func (c *fakeCatalogProbeAuthClient) ResolveEgressRequestHeaders(_ context.Context, request *authv1.ResolveEgressRequestHeadersRequest, _ ...grpc.CallOption) (*authv1.ResolveEgressRequestHeadersResponse, error) {
	c.resolveRequest = request
	return c.headers, c.err
}

type fakeCatalogProbeHeaderResolver struct {
	last    CatalogProbeHeaderRequest
	headers http.Header
	err     error
}

func (r *fakeCatalogProbeHeaderResolver) ResolveCatalogProbeHeaders(_ context.Context, request CatalogProbeHeaderRequest) (http.Header, error) {
	r.last = request
	return r.headers, r.err
}
