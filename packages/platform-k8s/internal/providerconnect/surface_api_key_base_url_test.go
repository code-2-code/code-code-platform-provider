package providerconnect

import "testing"

func TestResolveSurfaceAPIKeyBaseURLNoTemplate(t *testing.T) {
	t.Parallel()

	baseURL, usesOverride, err := resolveSurfaceAPIKeyBaseURL("https://api.openai.com/v1", "")
	if err != nil {
		t.Fatalf("resolveSurfaceAPIKeyBaseURL() error = %v", err)
	}
	if got, want := baseURL, "https://api.openai.com/v1"; got != want {
		t.Fatalf("base_url = %q, want %q", got, want)
	}
	if usesOverride {
		t.Fatal("usesOverride = true, want false")
	}
}

func TestResolveSurfaceAPIKeyBaseURLTemplateRequiresProvidedURL(t *testing.T) {
	t.Parallel()

	_, _, err := resolveSurfaceAPIKeyBaseURL(
		"https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1",
		"",
	)
	if err == nil {
		t.Fatal("resolveSurfaceAPIKeyBaseURL() error = nil, want validation error")
	}
}

func TestResolveSurfaceAPIKeyBaseURLTemplateAcceptsResolvedURL(t *testing.T) {
	t.Parallel()

	baseURL, usesOverride, err := resolveSurfaceAPIKeyBaseURL(
		"https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1",
		"https://api.cloudflare.com/client/v4/accounts/04d289f3ff972711c415793f0b7da61d/ai/v1",
	)
	if err != nil {
		t.Fatalf("resolveSurfaceAPIKeyBaseURL() error = %v", err)
	}
	if got, want := baseURL, "https://api.cloudflare.com/client/v4/accounts/04d289f3ff972711c415793f0b7da61d/ai/v1"; got != want {
		t.Fatalf("base_url = %q, want %q", got, want)
	}
	if !usesOverride {
		t.Fatal("usesOverride = false, want true")
	}
}

func TestResolveSurfaceAPIKeyBaseURLTemplateRejectsUnresolvedValue(t *testing.T) {
	t.Parallel()

	_, _, err := resolveSurfaceAPIKeyBaseURL(
		"https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1",
		"https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1",
	)
	if err == nil {
		t.Fatal("resolveSurfaceAPIKeyBaseURL() error = nil, want validation error")
	}
}
