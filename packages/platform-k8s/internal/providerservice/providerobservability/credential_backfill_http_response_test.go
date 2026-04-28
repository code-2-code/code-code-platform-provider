package providerobservability

import (
	"net/http"
	"testing"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
)

func TestCredentialBackfillValuesFromHTTPResponseExtractsDeclaredCookie(t *testing.T) {
	headers := http.Header{}
	headers.Add("Set-Cookie", "authjs.session-token=token-1; Path=/")
	headers.Add("Set-Cookie", "ignored=value; Path=/")
	headers.Add("Set-Cookie", "authjs.session-token=token-2; Path=/")

	values := credentialBackfillValuesFromHTTPResponse([]CredentialBackfillRule{{
		RuleID:     "authjs-session-token",
		Source:     observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_HTTP_RESPONSE_COOKIE,
		SourceName: "authjs.session-token",
	}}, headers)

	if got, want := values["authjs.session-token"], "token-2"; got != want {
		t.Fatalf("authjs.session-token = %q, want %q", got, want)
	}
	if _, exists := values["ignored"]; exists {
		t.Fatalf("ignored cookie was returned")
	}
}

func TestCredentialBackfillValuesFromHTTPResponseExtractsDeclaredHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Refresh-Token", "token-1")

	values := credentialBackfillValuesFromHTTPResponse([]CredentialBackfillRule{{
		RuleID:     "refresh-token",
		Source:     observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_HTTP_RESPONSE_HEADER,
		SourceName: "x-refresh-token",
	}}, headers)

	if got, want := values["x-refresh-token"], "token-1"; got != want {
		t.Fatalf("x-refresh-token = %q, want %q", got, want)
	}
}
