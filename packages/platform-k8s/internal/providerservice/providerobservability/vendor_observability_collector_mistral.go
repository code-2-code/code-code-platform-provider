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
	// mistralBillingTokensMetric records token usage from the Mistral billing
	// API, labelled by model_id and token_type.
	mistralBillingTokensMetric = providerUsageTokensMetric

	mistralBillingCollectorID = "mistral-billing"

	// mistralBillingURL is the Mistral console billing endpoint that requires a
	// web session token (obtainable by logging in to console.mistral.ai).
	mistralBillingURL = "https://console.mistral.ai/billing/v2/usage"
)

func init() {
	registerVendorObservabilityCollectorFactory(mistralBillingCollectorID, NewMistralVendorObservabilityCollector)
}

// NewMistralVendorObservabilityCollector returns a collector that probes the
// Mistral console billing/v2/usage endpoint using a management-plane session
// management-plane token resolved from account override or vendor fallback credential.
func NewMistralVendorObservabilityCollector() VendorObservabilityCollector {
	return &mistralVendorObservabilityCollector{}
}

type mistralVendorObservabilityCollector struct{}

func (c *mistralVendorObservabilityCollector) CollectorID() string {
	return mistralBillingCollectorID
}

func (c *mistralVendorObservabilityCollector) AuthAdapterID() string {
	return egressauth.AuthAdapterBearerSessionID
}

func (c *mistralVendorObservabilityCollector) Collect(ctx context.Context, input VendorObservabilityCollectInput) (*VendorObservabilityCollectResult, error) {
	token := observabilityCredentialToken(input.ObservabilityCredential)
	if token == "" {
		return nil, unauthorizedVendorObservabilityError("mistral billing: observability token is empty; configure provider observability authentication")
	}
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: http client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mistralBillingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: mistral billing: execute request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, vendorObservabilityMaxBodyReadSize))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, unauthorizedVendorObservabilityError(
			fmt.Sprintf("mistral billing: unauthorized: status %d %s", resp.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("providerobservability: mistral billing: failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	rows, err := parseMistralBillingGaugeRows(body)
	if err != nil {
		return nil, err
	}
	return &VendorObservabilityCollectResult{GaugeRows: rows}, nil
}
