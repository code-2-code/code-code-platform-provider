package providerobservability

import "testing"

func TestOAuthObservabilityOutcomeForAuthMessageClassifiesUnauthorized(t *testing.T) {
	t.Parallel()

	outcome := oauthObservabilityOutcomeForAuthMessage("antigravity request unauthorized: status 401")
	if outcome != ProbeOutcomeAuthBlocked {
		t.Fatalf("outcome = %q, want %q", outcome, ProbeOutcomeAuthBlocked)
	}
}
