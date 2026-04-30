package providerobservability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteHTTPProbeDoesNotFollowRedirect(t *testing.T) {
	t.Parallel()

	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer origin.Close()

	result, err := executeHTTPProbe(context.Background(), httpProbeSpec{
		CollectorName: "redirect test",
		URL:           origin.URL,
		HTTPClient:    target.Client(),
	})
	if err == nil {
		t.Fatal("executeHTTPProbe() error = nil, want redirect status error")
	}
	if result == nil || result.StatusCode != http.StatusFound {
		t.Fatalf("result status = %v, want %d", result, http.StatusFound)
	}
	if redirected {
		t.Fatal("executeHTTPProbe followed redirect")
	}
}
