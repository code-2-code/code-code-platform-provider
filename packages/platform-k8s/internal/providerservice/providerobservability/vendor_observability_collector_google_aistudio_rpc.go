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
	BaseURL          string
	Method           string
	Origin           string
	ProjectPath      string
	MetricTimeSeries googleAIStudioMetricTimeSeriesRequest
	Auth             googleAIStudioAuthInput
}

func (c *googleAIStudioObservabilityCollector) call(
	ctx context.Context,
	httpClient *http.Client,
	input googleAIStudioRPCCallInput,
) (responseBody []byte, err error) {
	ctx, span := startSurfaceObservabilityRPCSpan(ctx, "google_ai_studio", input.Method, http.MethodPost)
	defer func() {
		finishSurfaceObservabilityRPCSpan(span, err)
		span.End()
	}()
	payload, err := googleAIStudioRequestPayload(input)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(googleAIStudioRPCBaseURL)
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/" + strings.TrimSpace(input.Method)
	probeRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: create %s request: %w", input.Method, err)
	}
	probeRequest.Header.Set("Content-Type", "application/json+protobuf")
	probeRequest.Header.Set("Origin", strings.TrimSpace(input.Origin))
	probeRequest.Header.Set("Referer", strings.TrimRight(strings.TrimSpace(input.Origin), "/")+"/")
	probeRequest.Header.Set("User-Agent", googleAIStudioUserAgent)
	probeRequest.Header.Set("X-Goog-AuthUser", googleAIStudioRequestAuthUser)
	probeRequest.Header.Set("X-Goog-Encode-Response-If-Executable", "base64")
	probeRequest.Header.Set("X-User-Agent", "grpc-web-javascript/0.1")
	if err := applyGoogleAIStudioAuthHeaders(ctx, probeRequest, input.Auth); err != nil {
		return nil, err
	}

	recordSurfaceObservabilityRPCHost(span, probeRequest.URL.Host)
	recordSurfaceObservabilityHeaderFingerprint(span, "x-goog-authuser", probeRequest.Header.Get("X-Goog-AuthUser"))
	recordSurfaceObservabilityHeaderFingerprint(span, "x-user-agent", probeRequest.Header.Get("X-User-Agent"))
	payloadAttributes := []attribute.KeyValue{
		attribute.Bool("code_code.observability.project_path.present", strings.TrimSpace(input.ProjectPath) != ""),
		attribute.Int("http.request.body.size", len(payload)),
	}
	if input.MetricTimeSeries.ResourceCode > 0 {
		payloadAttributes = append(payloadAttributes,
			attribute.Int("code_code.observability.quota_metric.resource_code", input.MetricTimeSeries.ResourceCode),
			attribute.Int("code_code.observability.quota_metric.series_code", input.MetricTimeSeries.SeriesCode),
			attribute.Int("code_code.observability.quota_metric.tier_code", input.MetricTimeSeries.TierCode),
		)
	}
	recordSurfaceObservabilityRPCPayloadShape(span, payloadAttributes...)

	resp, err := httpClient.Do(probeRequest)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: execute %s request: %w", input.Method, err)
	}
	defer resp.Body.Close()
	recordSurfaceObservabilityHTTPResponse(span, resp.StatusCode, len(resp.Header.Values("Set-Cookie")))

	body, _ := io.ReadAll(io.LimitReader(resp.Body, observabilityMaxBodyReadSize))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, unauthorizedObservabilityError(
			fmt.Sprintf("google ai studio quotas: %s unauthorized: status %d", input.Method, resp.StatusCode),
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
