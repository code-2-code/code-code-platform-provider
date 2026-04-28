package providerobservability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"code-code.internal/platform-k8s/internal/egressauth"
)

func TestWithObservabilityEgressAuthUsesProviderSurfaceBindingID(t *testing.T) {
	t.Parallel()

	var requestHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestHeaders = r.Header.Clone()
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := withObservabilityEgressAuth(server.Client(), observabilityEgressAuth{
		ProviderSurfaceBindingID: "surface-1",
		RequestHeaderName:        "authorization",
		HeaderValuePrefix:        "Bearer",
		AuthAdapterID:            egressauth.AuthAdapterDefaultID,
	})
	request, err := http.NewRequest(http.MethodGet, server.URL+"/metrics", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	_ = response.Body.Close()

	if got := requestHeaders.Get("Authorization"); got != "" {
		t.Fatalf("authorization = %q, want empty", got)
	}
	if got, want := requestHeaders.Get(egressauth.HeaderProviderSurfaceBindingID), "surface-1"; got != want {
		t.Fatalf("%s = %q, want %q", egressauth.HeaderProviderSurfaceBindingID, got, want)
	}
}
