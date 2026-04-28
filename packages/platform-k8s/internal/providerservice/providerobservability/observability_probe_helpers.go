package providerobservability

import (
	"context"
	"net/http"
	"strings"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	"code-code.internal/platform-k8s/internal/platform/outboundhttp"
)

// observabilityHTTPClient creates an HTTP client with egress auth for
// observability probes. Shared by both vendor and OAuth runners.
func observabilityHTTPClient(ctx context.Context, auth observabilityEgressAuth) (*http.Client, error) {
	client, err := outboundhttp.NewClientFactory().NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return withObservabilityEgressAuth(client, auth), nil
}

// readCredentialMaterialFields reads credential material fields using the
// given reader and policy reference. Shared by both vendor and OAuth runners.
func readCredentialMaterialFields(
	ctx context.Context,
	reader CredentialMaterialReader,
	credentialID string,
	policyRef *authv1.CredentialMaterialReadPolicyRef,
	fields []string,
) (map[string]string, error) {
	credentialID = strings.TrimSpace(credentialID)
	if reader == nil || credentialID == "" || len(fields) == 0 {
		return nil, nil
	}
	values, err := reader.ReadCredentialMaterialFields(ctx, credentialID, policyRef, fields)
	if err != nil {
		return nil, err
	}
	return trimStringMap(values), nil
}

func trimStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
