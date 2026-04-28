package providerobservability

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code-code.internal/platform-k8s/internal/egressauth"
)

const (
	// meituanTokenUsageMetric records token usage from the LongCat console
	// tokenUsage API, labelled by model_id and token_type.
	meituanTokenUsageMetric = providerUsageTokensMetric

	meituanLongcatCollectorID = "meituan-longcat-token-usage"

	// meituanTokenUsageURL is the LongCat console token usage endpoint.
	// It requires a web session token obtained by logging in to longcat.chat.
	meituanTokenUsageURL = "https://longcat.chat/api/lc-platform/v1/tokenUsage?day=today"
)

func init() {
	registerVendorCollectorFactory(meituanLongcatCollectorID, NewMeituanLongcatObservabilityCollector)
}

// NewMeituanLongcatObservabilityCollector returns a collector that probes
// the LongCat console tokenUsage endpoint using a management-plane session
// management-plane token resolved from account override or vendor fallback credential.
func NewMeituanLongcatObservabilityCollector() ObservabilityCollector {
	return &meituanLongcatObservabilityCollector{}
}

type meituanLongcatObservabilityCollector struct{}

func (c *meituanLongcatObservabilityCollector) CollectorID() string {
	return meituanLongcatCollectorID
}

func (c *meituanLongcatObservabilityCollector) AuthAdapterID() string {
	return egressauth.AuthAdapterBearerSessionID
}

func (c *meituanLongcatObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (*ObservabilityCollectResult, error) {
	token := observabilityCredentialToken(input.ObservabilityCredential)
	if token == "" {
		return nil, unauthorizedObservabilityError("meituan longcat token usage: observability token is empty; configure provider observability authentication")
	}
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: http client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meituanTokenUsageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: execute request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, observabilityMaxBodyReadSize))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, unauthorizedObservabilityError(
			fmt.Sprintf("meituan longcat token usage: unauthorized: status %d %s", resp.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("providerobservability: meituan longcat token usage: failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	rows, err := parseMeituanTokenUsageGaugeRows(body)
	if err != nil {
		return nil, err
	}
	return &ObservabilityCollectResult{GaugeRows: rows}, nil
}
