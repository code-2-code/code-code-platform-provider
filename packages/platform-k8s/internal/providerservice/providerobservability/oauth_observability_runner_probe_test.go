package providerobservability

import "testing"

func TestOAuthObservabilityOutcomeForAuthMessageClassifiesUnauthorized(t *testing.T) {
	t.Parallel()

	outcome := oauthObservabilityOutcomeForAuthMessage("antigravity request unauthorized: status 401")
	if outcome != OAuthObservabilityProbeOutcomeAuthBlocked {
		t.Fatalf("outcome = %q, want %q", outcome, OAuthObservabilityProbeOutcomeAuthBlocked)
	}
}
