package providerobservability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

const googleAIStudioUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"

type googleAIStudioRPCCallInput struct {
	Method           string
	Authorization    string
	AuthUser         string
	PageAPIKey       string
	CookieHeader     string
	Origin           string
	ProjectPath      string
	MetricTimeSeries googleAIStudioMetricTimeSeriesRequest
}

func (c *googleAIStudioVendorObservabilityCollector) call(
	ctx context.Context,
	httpClient *http.Client,
	input googleAIStudioRPCCallInput,
) (responseBody []byte, err error) {
	ctx, span := startVendorObservabilityRPCSpan(ctx, "google_ai_studio", input.Method, http.MethodPost)
	defer func() {
		finishVendorObservabilityRPCSpan(span, err)
		span.End()
	}()
	payload, err := googleAIStudioRequestPayload(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(googleAIStudioRPCBaseURL, "/")+"/"+strings.TrimSpace(input.Method),
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: create %s request: %w", input.Method, err)
	}
	requestCookieHeader := strings.TrimSpace(input.CookieHeader)
	req.Header.Set("Authorization", strings.TrimSpace(input.Authorization))
	req.Header.Set("Content-Type", "application/json+protobuf")
	req.Header.Set("Cookie", requestCookieHeader)
	req.Header.Set("Origin", strings.TrimSpace(input.Origin))
	req.Header.Set("Referer", strings.TrimRight(strings.TrimSpace(input.Origin), "/")+"/")
	req.Header.Set("User-Agent", googleAIStudioUserAgent)
	req.Header.Set("X-Goog-Api-Key", strings.TrimSpace(input.PageAPIKey))
	req.Header.Set("X-Goog-AuthUser", googleAIStudioRequestAuthUser(input.AuthUser))
	req.Header.Set("X-Goog-Encode-Response-If-Executable", "base64")
	req.Header.Set("X-User-Agent", "grpc-web-javascript/0.1")

	recordVendorObservabilityRPCHost(span, req.URL.Host)
	recordVendorObservabilityHeaderFingerprint(span, "authorization", req.Header.Get("Authorization"))
	recordVendorObservabilityHeaderFingerprint(span, "cookie", req.Header.Get("Cookie"))
	recordVendorObservabilityHeaderFingerprint(span, "x-goog-api-key", req.Header.Get("X-Goog-Api-Key"))
	recordVendorObservabilityHeaderFingerprint(span, "x-goog-authuser", req.Header.Get("X-Goog-AuthUser"))
	recordVendorObservabilityHeaderFingerprint(span, "x-user-agent", req.Header.Get("X-User-Agent"))
	payloadAttributes := []attribute.KeyValue{
		attribute.Bool("code_code.observability.project_path.present", strings.TrimSpace(input.ProjectPath) != ""),
		attribute.Int("http.request.body.size", len(payload)),
	}
	if input.MetricTimeSeries.ResourceCode > 0 {
		payloadAttributes = append(payloadAttributes,
			attribute.String("code_code.observability.quota_metric.quota_type", input.MetricTimeSeries.QuotaType),
			attribute.Int("code_code.observability.quota_metric.resource_code", input.MetricTimeSeries.ResourceCode),
			attribute.Int("code_code.observability.quota_metric.series_code", input.MetricTimeSeries.SeriesCode),
			attribute.Int("code_code.observability.quota_metric.tier_code", input.MetricTimeSeries.TierCode),
		)
	}
	recordVendorObservabilityRPCPayloadShape(span, payloadAttributes...)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: execute %s request: %w", input.Method, err)
	}
	defer resp.Body.Close()
	recordVendorObservabilityHTTPResponse(span, resp.StatusCode, len(resp.Header.Values("Set-Cookie")))

	body, _ := io.ReadAll(io.LimitReader(resp.Body, vendorObservabilityMaxBodyReadSize))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, unauthorizedVendorObservabilityError(
			fmt.Sprintf("google ai studio quotas: %s unauthorized: status %d: %s", input.Method, resp.StatusCode, strings.TrimSpace(string(body))),
		)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"providerobservability: google ai studio quotas: %s failed with status %d: %s",
			input.Method,
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}
	return body, nil
}

func googleAIStudioRequestPayload(input googleAIStudioRPCCallInput) ([]byte, error) {
	switch strings.TrimSpace(input.Method) {
	case "ListCloudProjects", "ListQuotaModels":
		return []byte("[]"), nil
	case "ListModelRateLimits":
		path := strings.TrimSpace(input.ProjectPath)
		if path == "" {
			return nil, fmt.Errorf("providerobservability: google ai studio quotas: ListModelRateLimits project path is required")
		}
		return json.Marshal([]string{path})
	case "FetchMetricTimeSeries":
		return googleAIStudioMetricTimeSeriesPayload(input.ProjectPath, input.MetricTimeSeries)
	default:
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: unsupported rpc method %q", input.Method)
	}
}
