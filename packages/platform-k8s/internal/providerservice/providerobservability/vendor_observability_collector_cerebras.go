package providerobservability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code-code.internal/platform-k8s/internal/egressauth"
)

const (
	cerebrasQuotaLimitMetric        = providerQuotaLimitMetric
	cerebrasQuotaUsageMetric        = providerQuotaUsageMetric
	cerebrasQuotaRemainingMetric    = providerQuotaRemainingMetric
	cerebrasQuotaUsagePercentMetric = providerQuotaUsageFractionPercentMetric

	cerebrasCollectorID = "cerebras-quotas"

	cerebrasGraphQLURL    = "https://cloud.cerebras.ai/api/graphql"
	cerebrasSessionCookie = "authjs.session-token"
)

func init() {
	registerVendorObservabilityCollectorFactory(cerebrasCollectorID, NewCerebrasVendorObservabilityCollector)
}

// NewCerebrasVendorObservabilityCollector returns a collector that probes
// Cerebras quota and usage data via the Cloud console GraphQL API.
// Requires one management-plane session token resolved from account override or vendor fallback credential.
func NewCerebrasVendorObservabilityCollector() VendorObservabilityCollector {
	return &cerebrasVendorObservabilityCollector{}
}

type cerebrasVendorObservabilityCollector struct{}

func (c *cerebrasVendorObservabilityCollector) CollectorID() string {
	return cerebrasCollectorID
}

func (c *cerebrasVendorObservabilityCollector) AuthAdapterID() string {
	return egressauth.AuthAdapterSessionCookieID
}

func (c *cerebrasVendorObservabilityCollector) Collect(ctx context.Context, input VendorObservabilityCollectInput) (*VendorObservabilityCollectResult, error) {
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: http client is nil")
	}

	sessionToken := observabilitySessionValue(
		input.ObservabilityCredential,
		"authjs.session-token",
		"authjs_session_token",
		"session_token",
	)
	if sessionToken == "" {
		sessionToken = observabilityCredentialToken(input.ObservabilityCredential)
	}
	if sessionToken == "" {
		return nil, unauthorizedVendorObservabilityError("cerebras quotas: session token is required; configure provider observability authentication")
	}

	rows, backfillValues, err := cerebrasCollectGraphQL(ctx, input.HTTPClient, sessionToken, input.CredentialBackfills)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: no quota data collected")
	}
	return &VendorObservabilityCollectResult{
		GaugeRows:                rows,
		CredentialBackfillValues: backfillValues,
	}, nil
}

// cerebrasCollectGraphQL fetches quota limits and current usage via the
// Cerebras Cloud console GraphQL API.
func cerebrasCollectGraphQL(ctx context.Context, httpClient *http.Client, sessionToken string, backfillRules []CredentialBackfillRule) ([]VendorObservabilityMetricRow, map[string]string, error) {
	latestToken := strings.TrimSpace(sessionToken)

	organizations, backfillValues, err := cerebrasResolveOrganizations(ctx, httpClient, latestToken, backfillRules)
	if err != nil {
		return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: resolve organizations: %w", err)
	}

	rows := make([]VendorObservabilityMetricRow, 0)
	orgErrs := make([]string, 0)
	for _, org := range organizations {
		quotas, quotaBackfills, err := cerebrasGraphQLQuery(ctx, httpClient, latestToken, backfillRules,
			cerebrasListOrganizationEffectiveQuotasQuery,
			map[string]any{"organizationId": org.ID},
		)
		backfillValues = mergeCredentialBackfillValues(backfillValues, quotaBackfills)
		if err != nil {
			if isVendorObservabilityUnauthorizedError(err) {
				return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: list quotas for org %q: %w", org.ID, err)
			}
			orgErrs = append(orgErrs, fmt.Sprintf("%s: %s", org.ID, err.Error()))
			continue
		}

		usages, usageBackfills, err := cerebrasGraphQLQuery(ctx, httpClient, latestToken, backfillRules,
			cerebrasListOrganizationUsageQuery,
			map[string]any{"organizationId": org.ID},
		)
		backfillValues = mergeCredentialBackfillValues(backfillValues, usageBackfills)
		if err != nil {
			// Usage is best-effort; proceed with quotas only.
			usages = nil
		}
		orgRows, parseErr := parseCerebrasGraphQLGaugeRows(quotas, usages, org.ID, org.Name)
		if parseErr != nil {
			orgErrs = append(orgErrs, fmt.Sprintf("%s: %s", org.ID, parseErr.Error()))
			continue
		}
		rows = append(rows, orgRows...)
	}
	if len(rows) == 0 {
		if len(orgErrs) > 0 {
			return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: no metric rows returned: %s", strings.Join(orgErrs, "; "))
		}
		return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: no metric rows returned")
	}
	return rows, backfillValues, nil
}

// cerebrasResolveOrganizations calls ListMyOrganizations and returns all
// accessible organizations, preferring active ones first.
func cerebrasResolveOrganizations(ctx context.Context, httpClient *http.Client, sessionToken string, backfillRules []CredentialBackfillRule) ([]cerebrasOrganization, map[string]string, error) {
	body, backfillValues, err := cerebrasGraphQLQuery(ctx, httpClient, sessionToken, backfillRules,
		cerebrasListMyOrganizationsQuery,
		map[string]any{},
	)
	if err != nil {
		return nil, nil, err
	}
	organizations, parseErr := parseCerebrasOrganizations(body)
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return organizations, backfillValues, nil
}

// cerebrasGraphQLQuery executes one GraphQL query against the Cerebras Cloud
// console endpoint.
func cerebrasGraphQLQuery(ctx context.Context, httpClient *http.Client, sessionToken string, backfillRules []CredentialBackfillRule, query string, variables map[string]any) ([]byte, map[string]string, error) {
	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cerebrasGraphQLURL, bytes.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", cerebrasSessionCookie+"="+sessionToken)
	// Mimic browser origin to pass Cerebras server-side checks.
	req.Header.Set("Origin", "https://cloud.cerebras.ai")
	req.Header.Set("Referer", "https://cloud.cerebras.ai/platform")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:141.0) Gecko/20100101 Firefox/141.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: execute request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, vendorObservabilityMaxBodyReadSize))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil, unauthorizedVendorObservabilityError(
			fmt.Sprintf("cerebras quotas graphql: authjs.session-token is invalid or expired: status %d %s", resp.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, fmt.Errorf("providerobservability: cerebras quotas graphql: failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := cerebrasGraphQLResponseError(body); err != nil {
		return nil, nil, err
	}
	return body, credentialBackfillValuesFromHTTPResponse(backfillRules, resp.Header), nil
}

func cerebrasGraphQLResponseError(body []byte) error {
	type graphQLError struct {
		Message string `json:"message"`
	}
	var payload struct {
		Errors []graphQLError `json:"errors"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	if len(payload.Errors) == 0 {
		return nil
	}
	messages := make([]string, 0, len(payload.Errors))
	for _, item := range payload.Errors {
		message := strings.TrimSpace(item.Message)
		if message == "" {
			continue
		}
		messages = append(messages, message)
	}
	if len(messages) == 0 {
		return fmt.Errorf("providerobservability: cerebras quotas graphql: graphql returned errors")
	}
	joined := strings.Join(messages, "; ")
	lower := strings.ToLower(joined)
	if strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "expired") ||
		strings.Contains(lower, "logged in") ||
		strings.Contains(lower, "log in") ||
		strings.Contains(lower, "sign in") ||
		strings.Contains(lower, "session") ||
		strings.Contains(lower, "auth") {
		return unauthorizedVendorObservabilityError(
			fmt.Sprintf("cerebras quotas graphql: authjs.session-token is invalid or expired: %s", joined),
		)
	}
	return fmt.Errorf("providerobservability: cerebras quotas graphql: %s", joined)
}

// GraphQL query constants.
const cerebrasListMyOrganizationsQuery = `query ListMyOrganizations {
  ListMyOrganizations {
    id
    name
    organizationType
    state
    __typename
  }
}`

const cerebrasListOrganizationEffectiveQuotasQuery = `query ListOrganizationEffectiveQuotas($organizationId: ID!, $regionId: ID) {
  ListOrganizationEffectiveQuotas(organizationId: $organizationId, regionId: $regionId) {
    modelId
    regionId
    organizationId
    requestsPerMinute
    tokensPerMinute
    requestsPerHour
    tokensPerHour
    requestsPerDay
    tokensPerDay
    totalTokensPerMinute
    totalTokensPerHour
    totalTokensPerDay
    maxSequenceLength
    maxCompletionTokens
    __typename
  }
}`

const cerebrasListOrganizationUsageQuery = `query ListOrganizationUsage($organizationId: ID!) {
  ListOrganizationUsage(organizationId: $organizationId) {
    modelId
    regionId
    rpm
    tpm
    rph
    tph
    rpd
    tpd
    __typename
  }
}`
