package providerobservability

import (
	"strings"
	"testing"
)

func TestObservabilityUnauthorizedSafeMessageKeepsHeaderFailureContext(t *testing.T) {
	err := unauthorizedObservabilityError(`google ai studio quotas: resolve request auth headers failed: rpc error: code = FailedPrecondition desc = egress auth replacement failed for header "authorization"`)

	got := observabilityUnauthorizedSafeMessage(err)
	if !strings.Contains(got, `header "authorization"`) {
		t.Fatalf("safe message = %q, want authorization header context", got)
	}
}

func TestObservabilityUnauthorizedSafeMessageDropsUnauthorizedBody(t *testing.T) {
	err := unauthorizedObservabilityError("google ai studio quotas: ListCloudProjects unauthorized: status 401: upstream body with private details")

	got := observabilityUnauthorizedSafeMessage(err)
	if strings.Contains(got, "private details") {
		t.Fatalf("safe message = %q, want response body removed", got)
	}
	if !strings.Contains(got, "status 401") {
		t.Fatalf("safe message = %q, want status context", got)
	}
}
