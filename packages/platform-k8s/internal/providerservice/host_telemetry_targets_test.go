package providerservice

import (
	"testing"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func TestProviderHostTelemetryTargetGroupsDeduplicatesByHost(t *testing.T) {
	items := []*managementv1.ProviderSurfaceBindingView{
		apiSurface("https://API.OpenAI.com/v1/chat/completions?ignored=true"),
		apiSurface("https://api.openai.com/v1/models"),
		apiSurface("https://api.openai.com:443/other"),
		apiSurface("http://api.openai.com/v1"),
		apiSurface("https://api.openai.com:8443/v1"),
	}

	groups := providerHostTelemetryTargetGroups(items)

	if got, want := len(groups), 3; got != want {
		t.Fatalf("groups len = %d, want %d: %#v", got, want, groups)
	}
	assertTargetGroup(t, groups[0], "http://api.openai.com:80/", "http", "api.openai.com", "80")
	assertTargetGroup(t, groups[1], "https://api.openai.com:443/", "https", "api.openai.com", "443")
	assertTargetGroup(t, groups[2], "https://api.openai.com:8443/", "https", "api.openai.com", "8443")
}

func TestProviderHostTelemetryTargetGroupsSkipsNonAPIAndInvalidURLs(t *testing.T) {
	items := []*managementv1.ProviderSurfaceBindingView{
		apiSurface(""),
		apiSurface("mailto:ops@example.com"),
		{
			Runtime: &providerv1.ProviderSurfaceRuntime{
				Access: &providerv1.ProviderSurfaceRuntime_Cli{Cli: &providerv1.ProviderCLISurfaceRuntime{CliId: "codex"}},
			},
		},
		apiSurface("https://valid.example.com/v1"),
	}

	groups := providerHostTelemetryTargetGroups(items)

	if got, want := len(groups), 1; got != want {
		t.Fatalf("groups len = %d, want %d: %#v", got, want, groups)
	}
	assertTargetGroup(t, groups[0], "https://valid.example.com:443/", "https", "valid.example.com", "443")
}

func TestLimitProviderHostTelemetryTargetGroups(t *testing.T) {
	items := []prometheusHTTPDiscoveryTargetGroup{
		{Targets: []string{"https://a.example.com:443/"}},
		{Targets: []string{"https://b.example.com:443/"}},
	}

	limited := limitProviderHostTelemetryTargetGroups(items, 1)

	if got, want := len(limited), 1; got != want {
		t.Fatalf("limited len = %d, want %d", got, want)
	}
	if got, want := limited[0].Targets[0], "https://a.example.com:443/"; got != want {
		t.Fatalf("first target = %q, want %q", got, want)
	}
}

func TestLimitProviderHostTelemetryTargetGroupsAllowsUnlimited(t *testing.T) {
	items := []prometheusHTTPDiscoveryTargetGroup{
		{Targets: []string{"https://a.example.com:443/"}},
		{Targets: []string{"https://b.example.com:443/"}},
	}

	limited := limitProviderHostTelemetryTargetGroups(items, 0)

	if got, want := len(limited), 2; got != want {
		t.Fatalf("limited len = %d, want %d", got, want)
	}
}

func apiSurface(baseURL string) *managementv1.ProviderSurfaceBindingView {
	return &managementv1.ProviderSurfaceBindingView{
		Runtime: &providerv1.ProviderSurfaceRuntime{
			Access: &providerv1.ProviderSurfaceRuntime_Api{Api: &providerv1.ProviderAPISurfaceRuntime{
				Protocol: apiprotocolv1.Protocol_PROTOCOL_OPENAI_COMPATIBLE,
				BaseUrl:  baseURL,
			}},
		},
	}
}

func assertTargetGroup(t *testing.T, group prometheusHTTPDiscoveryTargetGroup, target, scheme, host, port string) {
	t.Helper()
	if got := group.Targets; len(got) != 1 || got[0] != target {
		t.Fatalf("targets = %#v, want [%q]", got, target)
	}
	if got := group.Labels["scheme"]; got != scheme {
		t.Fatalf("scheme label = %q, want %q", got, scheme)
	}
	if got := group.Labels["host"]; got != host {
		t.Fatalf("host label = %q, want %q", got, host)
	}
	if got := group.Labels["port"]; got != port {
		t.Fatalf("port label = %q, want %q", got, port)
	}
}
