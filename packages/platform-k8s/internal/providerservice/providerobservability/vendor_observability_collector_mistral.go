package providerobservability

import (
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// mistralBillingTokensMetric records token usage from the Mistral Admin
	// billing API, labelled by model_id and token_type.
	mistralBillingTokensMetric = providerUsageTokensMetric
	mistralBillingCostMetric   = providerUsageCostMetric
	mistralOTelProviderName    = "mistral_ai"

	mistralBillingCollectorID = "mistral-billing"

	mistralBillingOrigin  = "https://admin.mistral.ai"
	mistralBillingReferer = "https://admin.mistral.ai/organization/usage"
	mistralBillingPath    = "/api/billing/v2/usage"
	mistralLimitsReferer  = "https://admin.mistral.ai/plateforme/limits"
	mistralLimitsPath     = "/api/billing/limits"

	mistralAdminSessionPolicyID = "vendor.mistral-admin-session"
)

func init() {
	registerVendorCollectorFactory(mistralBillingCollectorID, NewMistralObservabilityCollector)
}

// NewMistralObservabilityCollector returns a collector that probes the Mistral
// Admin browser-session billing usage endpoint.
func NewMistralObservabilityCollector() ObservabilityCollector {
	return &mistralObservabilityCollector{now: time.Now}
}

type mistralObservabilityCollector struct {
	now func() time.Time
}

func (c *mistralObservabilityCollector) CollectorID() string {
	return mistralBillingCollectorID
}

func (c *mistralObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	billingURL := c.billingURL()
	billingHeaders, err := c.adminHeaders(ctx, input, billingURL, http.MethodGet, mistralBillingReferer)
	if err != nil {
		return nil, err
	}
	usageResult, err := executeHTTPProbe(ctx, httpProbeSpec{
		CollectorName: "mistral billing",
		URL:           billingURL,
		Method:        http.MethodGet,
		HTTPClient:    input.HTTPClient,
		ExtraHeaders:  billingHeaders,
	})
	if err != nil {
		if isMistralBillingAuthRedirect(usageResult) {
			return nil, unauthorizedObservabilityError("mistral billing: session redirected to login")
		}
		return nil, err
	}
	billingRows, err := parseMistralBillingGaugeRows(usageResult.Body)
	if err != nil {
		return nil, err
	}
	limitsURL := c.limitsURL()
	limitsHeaders, err := c.adminHeaders(ctx, input, limitsURL, http.MethodGet, mistralLimitsReferer)
	if err != nil {
		return nil, err
	}
	limitResult, err := executeHTTPProbe(ctx, httpProbeSpec{
		CollectorName: "mistral limits",
		URL:           limitsURL,
		Method:        http.MethodGet,
		HTTPClient:    input.HTTPClient,
		ExtraHeaders:  limitsHeaders,
	})
	if err != nil {
		if isMistralBillingAuthRedirect(limitResult) {
			return nil, unauthorizedObservabilityError("mistral limits: session redirected to login")
		}
		return nil, err
	}
	limitRows, err := parseMistralBillingLimitsGaugeRows(limitResult.Body)
	if err != nil {
		return nil, err
	}
	rows := append(billingRows, limitRows...)
	return &ObservabilityCollectResult{GaugeRows: rows}, nil
}

func (c *mistralObservabilityCollector) billingURL() string {
	now := time.Now().UTC()
	if c != nil && c.now != nil {
		now = c.now().UTC()
	}
	u, _ := url.Parse(mistralBillingOrigin)
	u.Path = mistralBillingPath
	q := u.Query()
	q.Set("month", fmt.Sprintf("%d", int(now.Month())))
	q.Set("year", fmt.Sprintf("%d", now.Year()))
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *mistralObservabilityCollector) limitsURL() string {
	u, _ := url.Parse(mistralBillingOrigin)
	u.Path = mistralLimitsPath
	return u.String()
}

func (c *mistralObservabilityCollector) adminHeaders(ctx context.Context, input ObservabilityCollectInput, rawURL string, method string, referer string) (map[string]string, error) {
	headers := map[string]string{
		"Accept":  "*/*",
		"Origin":  mistralBillingOrigin,
		"Referer": strings.TrimSpace(referer),
	}
	authHeaders, err := c.resolveAdminAuthHeaders(ctx, input, rawURL, method, headers)
	if err != nil {
		return nil, err
	}
	for name, value := range authHeaders {
		headers[name] = value
	}
	return headers, nil
}

func isMistralBillingAuthRedirect(result *httpProbeResult) bool {
	if result == nil || result.StatusCode < http.StatusMultipleChoices || result.StatusCode >= http.StatusBadRequest {
		return false
	}
	location := strings.TrimSpace(result.Headers.Get("Location"))
	if location == "" {
		return true
	}
	return strings.Contains(location, "auth.mistral.ai") || strings.Contains(location, "/self-service/login/")
}

func (c *mistralObservabilityCollector) resolveAdminAuthHeaders(ctx context.Context, input ObservabilityCollectInput, rawURL string, method string, requestHeaders map[string]string) (map[string]string, error) {
	if input.Auth == nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: auth client is not configured")
	}
	credentialID := strings.TrimSpace(input.CredentialID)
	if credentialID == "" {
		return nil, unauthorizedObservabilityError("mistral billing: observability credential is required")
	}
	policy, err := input.Auth.GetEgressAuthPolicy(ctx, &authv1.GetEgressAuthPolicyRequest{
		PolicyId: mistralAdminSessionPolicyID,
	})
	if err != nil {
		return nil, mistralBillingAuthError("read session auth policy", err)
	}
	if policy == nil || len(policy.GetRequestHeaderNames()) == 0 || len(policy.GetRequestReplacementRules()) == 0 {
		return nil, unauthorizedObservabilityError("mistral billing: session auth policy is unavailable")
	}
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: parse target url: %w", err)
	}
	response, err := input.Auth.ResolveEgressRequestHeaders(ctx, &authv1.ResolveEgressRequestHeadersRequest{
		PolicyId:               policy.GetPolicyId(),
		CredentialId:           credentialID,
		AdapterId:              policy.GetAdapterId(),
		TargetHost:             target.Host,
		TargetPath:             target.EscapedPath(),
		TargetMethod:           strings.TrimSpace(method),
		Origin:                 mistralBillingOrigin,
		RequestHeaders:         lowerHeaderMap(requestHeaders),
		AllowedHeaderNames:     policy.GetRequestHeaderNames(),
		SimpleReplacementRules: policy.GetRequestReplacementRules(),
	})
	if err != nil {
		return nil, mistralBillingAuthError("resolve request auth headers", err)
	}
	headers := mistralHeadersFromMutations(response.GetHeaders())
	if response.GetSkipped() || len(headers) == 0 {
		return nil, unauthorizedObservabilityError("mistral billing: request auth headers are unavailable")
	}
	return headers, nil
}

func mistralHeadersFromMutations(mutations []*authv1.EgressHeaderMutation) map[string]string {
	headers := map[string]string{}
	for _, mutation := range mutations {
		name := textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(mutation.GetName()))
		value := strings.TrimSpace(mutation.GetValue())
		if name == "" || value == "" {
			continue
		}
		headers[name] = value
	}
	return headers
}

func lowerHeaderMap(headers map[string]string) map[string]string {
	out := map[string]string{}
	for name, value := range headers {
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			continue
		}
		out[name] = value
	}
	return out
}

func mistralBillingAuthError(action string, err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.NotFound, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
		return unauthorizedObservabilityError(fmt.Sprintf("mistral billing: %s failed: %v", action, err))
	default:
		return fmt.Errorf("providerobservability: mistral billing: %s failed: %w", action, err)
	}
}
