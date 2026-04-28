package providerservice

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

const ProviderHostTelemetryTargetsPath = "/internal/prometheus/provider-host-targets"

type prometheusHTTPDiscoveryTargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type providerHostTelemetryTarget struct {
	key    string
	target string
	scheme string
	host   string
	port   string
}

func (s *Server) ProviderHostTelemetryTargetGroups(ctx context.Context) ([]prometheusHTTPDiscoveryTargetGroup, error) {
	items, err := s.providerSurfaceBindings.ListProviderSurfaceBindings(ctx)
	if err != nil {
		return nil, err
	}
	return limitProviderHostTelemetryTargetGroups(providerHostTelemetryTargetGroups(items), s.providerHostTargetLimit), nil
}

func (s *Server) ServeProviderHostTelemetryTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := s.ProviderHostTelemetryTargetGroups(r.Context())
	if err != nil {
		http.Error(w, "failed to build provider host targets", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		http.Error(w, "failed to encode provider host targets", http.StatusInternalServerError)
		return
	}
}

func providerHostTelemetryTargetGroups(items []*managementv1.ProviderSurfaceBindingView) []prometheusHTTPDiscoveryTargetGroup {
	targetsByKey := map[string]providerHostTelemetryTarget{}
	for _, item := range items {
		target, ok := providerHostTelemetryTargetFromSurface(item)
		if !ok {
			continue
		}
		targetsByKey[target.key] = target
	}
	targets := make([]providerHostTelemetryTarget, 0, len(targetsByKey))
	for _, target := range targetsByKey {
		targets = append(targets, target)
	}
	slices.SortFunc(targets, func(left, right providerHostTelemetryTarget) int {
		return strings.Compare(left.key, right.key)
	})
	groups := make([]prometheusHTTPDiscoveryTargetGroup, 0, len(targets))
	for _, target := range targets {
		groups = append(groups, prometheusHTTPDiscoveryTargetGroup{
			Targets: []string{target.target},
			Labels: map[string]string{
				"host":   target.host,
				"scheme": target.scheme,
				"port":   target.port,
			},
		})
	}
	return groups
}

func limitProviderHostTelemetryTargetGroups(items []prometheusHTTPDiscoveryTargetGroup, limit int) []prometheusHTTPDiscoveryTargetGroup {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

func providerHostTelemetryTargetFromSurface(item *managementv1.ProviderSurfaceBindingView) (providerHostTelemetryTarget, bool) {
	if item == nil || item.GetRuntime().GetApi() == nil {
		return providerHostTelemetryTarget{}, false
	}
	return normalizeProviderHostTelemetryTarget(item.GetRuntime().GetApi().GetBaseUrl())
}

func normalizeProviderHostTelemetryTarget(raw string) (providerHostTelemetryTarget, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return providerHostTelemetryTarget{}, false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return providerHostTelemetryTarget{}, false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return providerHostTelemetryTarget{}, false
	}
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(parsed.Hostname())), ".")
	if host == "" {
		return providerHostTelemetryTarget{}, false
	}
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		port = defaultPortForScheme(scheme)
	}
	if port == "" {
		return providerHostTelemetryTarget{}, false
	}
	targetURL := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
		Path:   "/",
	}
	key := strings.Join([]string{scheme, host, port}, "\x00")
	return providerHostTelemetryTarget{
		key:    key,
		target: targetURL.String(),
		scheme: scheme,
		host:   host,
		port:   port,
	}, true
}

func defaultPortForScheme(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}
