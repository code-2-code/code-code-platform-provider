package providerobservability

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const vendorObservabilityTraceMessageLimit = 500

var (
	vendorObservabilityTracer = otel.Tracer("code-code/platform-k8s/authservice/internal/credentials/vendor-observability")
)

func startVendorObservabilityProbeSpan(ctx context.Context, providerID string, providerSurfaceBindingID string, trigger VendorObservabilityProbeTrigger) (context.Context, trace.Span) {
	return vendorObservabilityTracer.Start(ctx, "vendor_observability.operation", trace.WithAttributes(
		attribute.String("code_code.provider.id", providerID),
		attribute.String("code_code.provider.surface_id", providerSurfaceBindingID),
		attribute.String("code_code.observability.trigger", string(trigger)),
	))
}

func finishVendorObservabilityProbeSpan(span trace.Span, result *VendorObservabilityProbeResult) {
	if span == nil || result == nil {
		return
	}

	span.SetAttributes(
		attribute.String("code_code.vendor.id", result.VendorID),
		attribute.String("code_code.provider.id", result.ProviderID),
		attribute.String("code_code.provider.surface_id", result.ProviderSurfaceBindingID),
		attribute.String("code_code.observability.outcome", string(result.Outcome)),
	)
	if result.Message != "" {
		message := safeTraceMessage(result.Message)
		span.AddEvent("vendor_observability.operation.message", trace.WithAttributes(
			attribute.String("code_code.observability.message", message),
		))
	}
	if result.Reason != "" {
		span.SetAttributes(attribute.String("code_code.observability.reason", result.Reason))
		if result.Outcome == VendorObservabilityProbeOutcomeAuthBlocked {
			span.SetAttributes(attribute.String("code_code.observability.auth_blocked.reason", result.Reason))
		}
	}

	switch result.Outcome {
	case VendorObservabilityProbeOutcomeAuthBlocked:
		reason := strings.TrimSpace(result.Reason)
		if reason == "" {
			reason = "auth_blocked"
		}
		recordVendorObservabilitySpanMessage(span, result.Message, reason)
	case VendorObservabilityProbeOutcomeFailed:
		reason := strings.TrimSpace(result.Reason)
		if reason == "" {
			reason = "probe_failed"
		}
		recordVendorObservabilitySpanMessage(span, result.Message, reason)
	}
}

func startVendorObservabilityCollectSpan(ctx context.Context, collectorID string) (context.Context, trace.Span) {
	return vendorObservabilityTracer.Start(ctx, "vendor_observability.collect", trace.WithAttributes(
		attribute.String("code_code.observability.collector_id", collectorID),
	))
}

func finishVendorObservabilityCollectSpan(span trace.Span, err error) {
	recordVendorObservabilitySpanError(span, err, "collect_failed")
}

func recordVendorObservabilityCredentialPresence(span trace.Span, field string, present bool) {
	if span == nil {
		return
	}
	span.SetAttributes(attribute.Bool("code_code.observability.credential."+safeTraceAttributeToken(field)+".present", present))
}

func startVendorObservabilityRPCSpan(ctx context.Context, system string, rpcMethod string, httpMethod string) (context.Context, trace.Span) {
	return vendorObservabilityTracer.Start(ctx, "vendor_observability.rpc", trace.WithAttributes(
		attribute.String("rpc.system", system),
		attribute.String("rpc.method", rpcMethod),
		attribute.String("http.request.method", httpMethod),
	))
}

func finishVendorObservabilityRPCSpan(span trace.Span, err error) {
	recordVendorObservabilitySpanError(span, err, "rpc_failed")
}

func recordVendorObservabilityRPCHost(span trace.Span, host string) {
	if span == nil || strings.TrimSpace(host) == "" {
		return
	}
	span.SetAttributes(attribute.String("server.address", host))
}

func recordVendorObservabilityHeaderPresence(span trace.Span, headerName string, present bool) {
	if span == nil {
		return
	}
	span.SetAttributes(attribute.Bool("http.request.header."+safeTraceAttributeToken(headerName)+".present", present))
}

func recordVendorObservabilityHeaderFingerprint(span trace.Span, headerName string, value string) {
	if span == nil {
		return
	}
	token := safeTraceAttributeToken(headerName)
	trimmed := strings.TrimSpace(value)
	span.SetAttributes(
		attribute.Bool("http.request.header."+token+".present", trimmed != ""),
		attribute.Int("http.request.header."+token+".length", len(trimmed)),
	)
	if trimmed == "" {
		return
	}
	digest := sha256.Sum256([]byte(trimmed))
	span.SetAttributes(attribute.String("http.request.header."+token+".sha256_12", fmt.Sprintf("%x", digest[:])[:12]))
}

func recordVendorObservabilityRPCPayloadShape(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil || len(attrs) == 0 {
		return
	}
	span.SetAttributes(attrs...)
}

func recordVendorObservabilityHTTPResponse(span trace.Span, statusCode int, setCookieCount int) {
	if span == nil {
		return
	}
	span.SetAttributes(
		attribute.Int("http.response.status_code", statusCode),
		attribute.Int("code_code.google_aistudio.set_cookie.count", setCookieCount),
	)
	if statusCode == 401 || statusCode == 403 {
		span.SetAttributes(attribute.String("code_code.observability.error_reason", "unauthorized_http_status"))
		span.SetStatus(codes.Error, fmt.Sprintf("unauthorized http status %d", statusCode))
		return
	}
	if statusCode < 200 || statusCode >= 300 {
		span.SetAttributes(attribute.String("code_code.observability.error_reason", "unexpected_http_status"))
		span.SetStatus(codes.Error, fmt.Sprintf("unexpected http status %d", statusCode))
	}
}

func (r *VendorObservabilityRunner) logVendorObservabilityProbeFailure(result *VendorObservabilityProbeResult) {
	if r == nil || r.logger == nil || result == nil {
		return
	}
	switch result.Outcome {
	case VendorObservabilityProbeOutcomeAuthBlocked, VendorObservabilityProbeOutcomeFailed:
	default:
		return
	}
	r.logger.Warn(
		"vendor observability operation failed",
		"vendor_id", result.VendorID,
		"provider_id", result.ProviderID,
		"provider_surface_binding_id", result.ProviderSurfaceBindingID,
		"outcome", string(result.Outcome),
		"reason", result.Reason,
		"message", safeTraceMessage(result.Message),
	)
}

func recordVendorObservabilitySpanMessage(span trace.Span, message string, reason string) {
	message = safeTraceMessage(message)
	if message == "" {
		message = reason
	}
	recordVendorObservabilitySpanError(span, errors.New(message), reason)
}

func recordVendorObservabilitySpanError(span trace.Span, err error, reason string) {
	if span == nil || err == nil {
		return
	}
	message := safeTraceMessage(err.Error())
	if reason != "" {
		span.SetAttributes(attribute.String("code_code.observability.error_reason", reason))
		span.RecordError(errors.New(message), trace.WithAttributes(attribute.String("code_code.observability.error_reason", reason)))
	} else {
		span.RecordError(errors.New(message))
	}
	span.SetStatus(codes.Error, message)
}

func safeTraceAttributeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	token := strings.Trim(builder.String(), "_")
	if token == "" {
		return "unknown"
	}
	return token
}

func safeTraceMessage(message string) string {
	message = strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if message == "" {
		return ""
	}
	runes := []rune(message)
	if len(runes) <= vendorObservabilityTraceMessageLimit {
		return message
	}
	return string(runes[:vendorObservabilityTraceMessageLimit]) + "..."
}
