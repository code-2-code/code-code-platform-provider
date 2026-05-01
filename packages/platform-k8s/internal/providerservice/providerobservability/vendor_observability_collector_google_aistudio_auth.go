package providerobservability

import (
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	googleAIStudioVendorID        = "google"
	googleAIStudioSessionPolicyID = "vendor.google-aistudio-session"
)

func (c *googleAIStudioObservabilityCollector) readProjectID(ctx context.Context, input ObservabilityCollectInput) (string, error) {
	if input.Auth == nil {
		return "", fmt.Errorf("providerobservability: google ai studio quotas: auth client is not configured")
	}
	credentialID := strings.TrimSpace(input.CredentialID)
	if credentialID == "" {
		return "", unauthorizedObservabilityError("google ai studio quotas: observability credential is required")
	}
	response, err := input.Auth.ReadCredentialMaterialFields(ctx, &authv1.ReadCredentialMaterialFieldsRequest{
		CredentialId: credentialID,
		FieldIds:     []string{materialKeyProjectID},
		PolicyRef: &authv1.CredentialMaterialReadPolicyRef{
			Kind:        authv1.CredentialMaterialReadPolicyKind_CREDENTIAL_MATERIAL_READ_POLICY_KIND_VENDOR_ACTIVE_QUERY,
			OwnerId:     googleAIStudioVendorID,
			SurfaceId:   strings.TrimSpace(input.SurfaceID),
			CollectorId: googleAIStudioCollectorID,
		},
	})
	if err != nil {
		return "", googleAIStudioAuthError("read project_id", err)
	}
	return strings.TrimSpace(response.GetValues()[materialKeyProjectID]), nil
}

func googleAIStudioAuthFields(input ObservabilityCollectInput) googleAIStudioAuthInput {
	return googleAIStudioAuthInput{
		Auth:         input.Auth,
		CredentialID: strings.TrimSpace(input.CredentialID),
	}
}

type googleAIStudioAuthInput struct {
	Auth         ObservabilityAuthClient
	CredentialID string
}

func applyGoogleAIStudioAuthHeaders(ctx context.Context, request *http.Request, input googleAIStudioAuthInput) error {
	if request == nil {
		return fmt.Errorf("providerobservability: google ai studio quotas: request is nil")
	}
	if input.Auth == nil {
		return fmt.Errorf("providerobservability: google ai studio quotas: auth client is not configured")
	}
	credentialID := strings.TrimSpace(input.CredentialID)
	if credentialID == "" {
		return unauthorizedObservabilityError("google ai studio quotas: observability credential is required")
	}
	policy, err := input.Auth.GetEgressAuthPolicy(ctx, &authv1.GetEgressAuthPolicyRequest{
		PolicyId: googleAIStudioSessionPolicyID,
	})
	if err != nil {
		return googleAIStudioAuthError("read session auth policy", err)
	}
	if policy == nil || len(policy.GetRequestHeaderNames()) == 0 || len(policy.GetRequestReplacementRules()) == 0 {
		return unauthorizedObservabilityError("google ai studio quotas: session auth policy is unavailable")
	}
	response, err := input.Auth.ResolveEgressRequestHeaders(ctx, &authv1.ResolveEgressRequestHeadersRequest{
		PolicyId:               policy.GetPolicyId(),
		CredentialId:           credentialID,
		AdapterId:              policy.GetAdapterId(),
		TargetHost:             request.URL.Host,
		TargetPath:             request.URL.EscapedPath(),
		TargetMethod:           request.Method,
		Origin:                 googleAIStudioOriginFromRequest(request),
		RequestHeaders:         googleAIStudioRequestHeaders(request),
		AllowedHeaderNames:     policy.GetRequestHeaderNames(),
		SimpleReplacementRules: policy.GetRequestReplacementRules(),
	})
	if err != nil {
		return googleAIStudioAuthError("resolve request auth headers", err)
	}
	headers := googleAIStudioHeadersFromMutations(response.GetHeaders())
	if response.GetSkipped() || len(headers) == 0 {
		return unauthorizedObservabilityError("google ai studio quotas: request auth headers are unavailable")
	}
	for name, values := range headers {
		request.Header.Del(name)
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	return nil
}

func googleAIStudioRequestHeaders(request *http.Request) map[string]string {
	headers := map[string]string{}
	for _, name := range []string{"origin", "referer"} {
		if value := strings.TrimSpace(request.Header.Get(name)); value != "" {
			headers[name] = value
		}
	}
	return headers
}

func googleAIStudioOriginFromRequest(request *http.Request) string {
	if request == nil {
		return ""
	}
	if origin := strings.TrimSpace(request.Header.Get("Origin")); origin != "" {
		return origin
	}
	referer := strings.TrimSpace(request.Header.Get("Referer"))
	if referer == "" {
		return ""
	}
	parsed, err := url.Parse(referer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func googleAIStudioHeadersFromMutations(mutations []*authv1.EgressHeaderMutation) http.Header {
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

func googleAIStudioAuthError(action string, err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.NotFound, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition:
		return unauthorizedObservabilityError(fmt.Sprintf("google ai studio quotas: %s failed: %v", action, err))
	default:
		return fmt.Errorf("providerobservability: google ai studio quotas: %s failed: %w", action, err)
	}
}
