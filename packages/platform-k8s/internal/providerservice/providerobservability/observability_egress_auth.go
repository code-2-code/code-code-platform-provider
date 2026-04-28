package providerobservability

import (
	"encoding/json"
	"net/http"
	"net/textproto"
	"sort"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/platform-k8s/internal/egressauth"
)

type observabilityEgressAuth struct {
	CLIID                    string
	VendorID                 string
	ProviderID               string
	ProviderSurfaceBindingID string
	RequestHeaderName        string
	HeaderValuePrefix        string
	AuthAdapterID            string
}

func withObservabilityEgressAuth(client *http.Client, auth observabilityEgressAuth) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	if strings.TrimSpace(auth.ProviderSurfaceBindingID) == "" {
		return client
	}
	next := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	next.Transport = observabilityEgressAuthRoundTripper{
		base: base,
		auth: auth,
	}
	return &next
}

type observabilityEgressAuthRoundTripper struct {
	base http.RoundTripper
	auth observabilityEgressAuth
}

func (t observabilityEgressAuthRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	next := request.Clone(request.Context())
	next.Header = request.Header.Clone()
	headerNames := replacementHeaderNames(next.Header)
	for _, name := range headerNames {
		next.Header.Del(textproto.CanonicalMIMEHeaderKey(name))
	}
	if len(headerNames) == 0 {
		headerName := strings.ToLower(strings.TrimSpace(t.auth.RequestHeaderName))
		if headerName == "" {
			headerName = "authorization"
		}
		headerNames = []string{headerName}
	}
	for _, name := range headerNames {
		next.Header.Del(textproto.CanonicalMIMEHeaderKey(name))
	}
	if host := strings.TrimSpace(next.URL.Host); host != "" {
		next.Header.Set(egressauth.HeaderTargetHosts, host)
	}
	next.Header.Set(egressauth.HeaderRequestHeaderNames, strings.Join(headerNames, ","))
	setHeaderRules(next.Header, headerNames, t.auth.HeaderValuePrefix)
	if prefix := strings.TrimSpace(t.auth.HeaderValuePrefix); prefix != "" {
		next.Header.Set(egressauth.HeaderHeaderValuePrefix, prefix)
	}
	setOptionalHeader(next.Header, egressauth.HeaderAuthAdapterID, t.auth.AuthAdapterID)
	setOptionalHeader(next.Header, egressauth.HeaderCLIID, t.auth.CLIID)
	setOptionalHeader(next.Header, egressauth.HeaderVendorID, t.auth.VendorID)
	setOptionalHeader(next.Header, egressauth.HeaderProviderID, t.auth.ProviderID)
	setOptionalHeader(next.Header, egressauth.HeaderProviderSurfaceBindingID, t.auth.ProviderSurfaceBindingID)
	return t.base.RoundTrip(next)
}

func replacementHeaderNames(headers http.Header) []string {
	if len(headers) == 0 {
		return nil
	}
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for name, values := range headers {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		for _, value := range values {
			if !strings.Contains(value, egressauth.Placeholder) {
				continue
			}
			if _, ok := seen[normalized]; ok {
				break
			}
			seen[normalized] = struct{}{}
			names = append(names, normalized)
			break
		}
	}
	sort.Strings(names)
	return names
}

func setOptionalHeader(headers http.Header, name string, value string) {
	if value = strings.TrimSpace(value); value != "" {
		headers.Set(name, value)
	}
}

func fakeOAuthCredential(credentialID string) *credentialv1.ResolvedCredential {
	return &credentialv1.ResolvedCredential{
		CredentialId: strings.TrimSpace(credentialID),
		Kind:         credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH,
		Material: &credentialv1.ResolvedCredential_Oauth{
			Oauth: &credentialv1.OAuthCredential{
				AccessToken: egressauth.Placeholder,
				TokenType:   "Bearer",
			},
		},
	}
}

func setHeaderRules(headers http.Header, headerNames []string, prefix string) {
	rules := simpleReplacementRulesForHeaderNames(headerNames, prefix)
	if len(rules) == 0 {
		return
	}
	payload, err := json.Marshal(rules)
	if err != nil {
		return
	}
	headers.Set(egressauth.HeaderRequestHeaderRulesJSON, string(payload))
}

func simpleReplacementRulesForHeaderNames(headerNames []string, prefix string) []egressauth.SimpleReplacementRule {
	out := make([]egressauth.SimpleReplacementRule, 0, len(headerNames))
	for _, name := range headerNames {
		name = strings.ToLower(strings.TrimSpace(name))
		rule := egressauth.SimpleReplacementRule{HeaderName: name}
		switch name {
		case "authorization":
			rule.Mode = egressauth.SimpleReplacementModeBearer
			if strings.TrimSpace(prefix) != "" {
				rule.HeaderValuePrefix = strings.TrimSpace(prefix)
			}
		case "x-goog-api-key":
			rule.Mode = egressauth.SimpleReplacementModeGoogleAPIKey
		case "x-api-key":
			rule.Mode = egressauth.SimpleReplacementModeXAPIKey
		case "cookie":
			rule.Mode = egressauth.SimpleReplacementModeCookie
		default:
			continue
		}
		out = append(out, egressauth.NormalizeSimpleReplacementRule(rule))
	}
	return out
}

func fakeSessionCredential(credentialID string) *credentialv1.ResolvedCredential {
	return &credentialv1.ResolvedCredential{
		CredentialId: strings.TrimSpace(credentialID),
		Kind:         credentialv1.CredentialKind_CREDENTIAL_KIND_SESSION,
		Material: &credentialv1.ResolvedCredential_Session{
			Session: &credentialv1.SessionCredential{
				Values: map[string]string{
					"access_token":         egressauth.Placeholder,
					"api_key":              egressauth.Placeholder,
					"authjs.session-token": egressauth.Placeholder,
					"authjs_session_token": egressauth.Placeholder,
					"authorization":        egressauth.Placeholder,
					"cookie":               egressauth.Placeholder,
					"page_api_key":         egressauth.Placeholder,
					"session_token":        egressauth.Placeholder,
					"token":                egressauth.Placeholder,
				},
			},
		},
	}
}
