package providerobservability

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// httpProbeSpec describes one observability HTTP probe request.
// Collectors pass vendor-specific parameters; the shared execution
// handles token validation, request creation, response reading,
// and status-code dispatch.
type httpProbeSpec struct {
	// CollectorName is used in error message prefixes, e.g. "mistral billing".
	CollectorName string
	URL           string
	Method        string    // defaults to GET
	Body          io.Reader // optional request body
	HTTPClient    *http.Client
	ExtraHeaders  map[string]string // additional headers (e.g. Content-Type)
	MaxBodySize   int64             // defaults to observabilityMaxBodyReadSize
}

// httpProbeResult carries the raw HTTP response for collector-specific parsing.
type httpProbeResult struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// executeHTTPProbe executes a single HTTP probe and handles the common
// boilerplate shared by most observability collectors:
//  1. httpClient nil check
//  2. request creation with Accept and caller-supplied non-auth headers
//  3. response body read with LimitReader
//  4. 401/403 → unauthorizedObservabilityError
//  5. non-2xx → generic error
func executeHTTPProbe(ctx context.Context, spec httpProbeSpec) (*httpProbeResult, error) {
	name := strings.TrimSpace(spec.CollectorName)
	if name == "" {
		name = "observability"
	}

	if spec.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: %s: http client is nil", name)
	}

	method := strings.TrimSpace(spec.Method)
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, spec.URL, spec.Body)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: %s: create request: %w", name, err)
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range spec.ExtraHeaders {
		req.Header.Set(key, value)
	}

	client := *spec.HTTPClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: %s: execute request: %w", name, err)
	}
	defer resp.Body.Close()

	maxBody := spec.MaxBodySize
	if maxBody <= 0 {
		maxBody = observabilityMaxBodyReadSize
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, unauthorizedObservabilityError(
			fmt.Sprintf("%s: unauthorized: status %d %s", name, resp.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// Return the result so callers with special status handling (e.g. 429)
		// can inspect it before treating as fatal.
		return &httpProbeResult{
			StatusCode: resp.StatusCode,
			Body:       body,
			Headers:    resp.Header,
		}, fmt.Errorf("providerobservability: %s: failed with status %d: %s", name, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return &httpProbeResult{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}
