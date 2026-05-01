package providercatalogs

import (
	"context"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	modelcatalogdiscoveryv1 "code-code.internal/go-contract/model_catalog_discovery/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"google.golang.org/grpc"
)

const catalogProbeAuthTimeout = 5 * time.Second

type egressAuthCatalogProbeHeaderResolver struct {
	auth egressAuthCatalogProbeClient
}

type egressAuthCatalogProbeClient interface {
	GetEgressAuthPolicy(context.Context, *authv1.GetEgressAuthPolicyRequest, ...grpc.CallOption) (*authv1.GetEgressAuthPolicyResponse, error)
	ResolveEgressRequestHeaders(context.Context, *authv1.ResolveEgressRequestHeadersRequest, ...grpc.CallOption) (*authv1.ResolveEgressRequestHeadersResponse, error)
}

func NewEgressAuthCatalogProbeHeaderResolver(auth egressAuthCatalogProbeClient) CatalogProbeHeaderResolver {
	if auth == nil {
		return nil
	}
	return &egressAuthCatalogProbeHeaderResolver{auth: auth}
}

func (r *egressAuthCatalogProbeHeaderResolver) ResolveCatalogProbeHeaders(ctx context.Context, request CatalogProbeHeaderRequest) (http.Header, error) {
	if r == nil || r.auth == nil {
		return nil, modelCatalogProbeAuthError("auth header resolver is not configured")
	}
	credentialID := strings.TrimSpace(request.CredentialID)
	if credentialID == "" {
		return nil, modelCatalogProbeAuthError("provider credential is required for secure model catalog probe")
	}
	credentialKind, ok, err := catalogProbeCredentialKind(request.Operation)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, catalogProbeAuthTimeout)
	defer cancel()
	policy, err := r.auth.GetEgressAuthPolicy(ctx, &authv1.GetEgressAuthPolicyRequest{
		CredentialKind: credentialKind,
		Protocol:       request.Protocol,
	})
	if err != nil {
		return nil, err
	}
	if policy == nil || len(policy.GetRequestHeaderNames()) == 0 || len(policy.GetRequestReplacementRules()) == 0 {
		return nil, modelCatalogProbeAuthError("request auth policy is unavailable")
	}
	targetHost, targetPath := catalogProbeTarget(request.BaseURL, request.Operation)
	response, err := r.auth.ResolveEgressRequestHeaders(ctx, &authv1.ResolveEgressRequestHeadersRequest{
		PolicyId:               policy.GetPolicyId(),
		CredentialId:           credentialID,
		AdapterId:              policy.GetAdapterId(),
		TargetHost:             targetHost,
		TargetPath:             targetPath,
		TargetMethod:           catalogProbeHTTPMethod(request.Operation),
		HeaderValuePrefix:      policy.GetHeaderValuePrefix(),
		SimpleReplacementRules: policy.GetRequestReplacementRules(),
		AllowedHeaderNames:     policy.GetRequestHeaderNames(),
	})
	if err != nil {
		return nil, err
	}
	headers := headersFromEgressMutations(response.GetHeaders())
	if response.GetSkipped() || len(headers) == 0 {
		return nil, modelCatalogProbeAuthError("request auth headers are unavailable")
	}
	return headers, nil
}

func catalogProbeCredentialKind(operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation) (credentialv1.CredentialKind, bool, error) {
	for _, requirement := range operation.GetSecurity() {
		for _, scheme := range requirement.GetSchemes() {
			switch scheme {
			case modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_ANONYMOUS:
				continue
			case modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_API_KEY:
				return credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY, true, nil
			case modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_OAUTH:
				return credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH, true, nil
			case modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_SESSION:
				return credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION, true, nil
			case modelcatalogdiscoveryv1.ModelCatalogDiscoverySecurityScheme_MODEL_CATALOG_DISCOVERY_SECURITY_SCHEME_UNSPECIFIED:
				return credentialv1.CredentialKind_CREDENTIAL_KIND_UNSPECIFIED, false, modelCatalogProbeAuthError("model catalog probe security scheme is unspecified")
			default:
				return credentialv1.CredentialKind_CREDENTIAL_KIND_UNSPECIFIED, false, modelCatalogProbeAuthError("unsupported model catalog probe security scheme %s", scheme.String())
			}
		}
	}
	return credentialv1.CredentialKind_CREDENTIAL_KIND_UNSPECIFIED, false, nil
}

func headersFromEgressMutations(mutations []*authv1.EgressHeaderMutation) http.Header {
	headers := http.Header{}
	for _, mutation := range mutations {
		name := textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(mutation.GetName()))
		value := strings.TrimSpace(mutation.GetValue())
		if name == "" || value == "" {
			continue
		}
		switch mutation.GetAppendAction() {
		case authv1.EgressHeaderAppendAction_EGRESS_HEADER_APPEND_ACTION_APPEND_IF_EXISTS_OR_ADD:
			headers.Add(name, value)
		case authv1.EgressHeaderAppendAction_EGRESS_HEADER_APPEND_ACTION_ADD_IF_ABSENT:
			if headers.Get(name) == "" {
				headers.Set(name, value)
			}
		default:
			headers.Set(name, value)
		}
	}
	return headers
}

func catalogProbeTarget(baseURL string, operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation) (string, string) {
	baseURL = strings.TrimSpace(baseURL)
	if override := strings.TrimSpace(operation.GetBaseUrl()); override != "" {
		baseURL = override
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed == nil {
		return "", ""
	}
	path := strings.TrimSpace(operation.GetPath())
	if path == "" {
		return strings.TrimSpace(parsed.Host), strings.TrimSpace(parsed.Path)
	}
	if strings.HasPrefix(path, "/") {
		return strings.TrimSpace(parsed.Host), path
	}
	return strings.TrimSpace(parsed.Host), strings.TrimRight(parsed.Path, "/") + "/" + path
}

func catalogProbeHTTPMethod(operation *modelcatalogdiscoveryv1.ModelCatalogDiscoveryOperation) string {
	if operation.GetMethod() == modelcatalogdiscoveryv1.DiscoveryHTTPMethod_DISCOVERY_HTTP_METHOD_POST {
		return http.MethodPost
	}
	return http.MethodGet
}

func modelCatalogProbeAuthError(format string, args ...any) *domainerror.ValidationError {
	return domainerror.NewValidation("platformk8s/providercatalogs: "+format, args...)
}
