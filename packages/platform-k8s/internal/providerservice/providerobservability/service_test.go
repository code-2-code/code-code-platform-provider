package providerobservability

import (
	"context"
	"errors"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

func TestServiceProbeProviderDispatchesCapability(t *testing.T) {
	capability := &fakeCapability{
		kind:           OwnerKindCLI,
		ownerID:        "codex",
		matchSurfaceID: "endpoint-a",
		result: &ProbeResult{
			OwnerKind:  OwnerKindCLI,
			OwnerID:    "codex",
			ProviderID: "provider-a",
			Outcome:    ProbeOutcomeExecuted,
			Message:    "ok",
		},
	}
	service, err := NewService(Config{
		ProviderSurfaceBindings: fakeSurfaceBindingLister{items: []*managementv1.ProviderSurfaceBindingView{
			{ProviderId: "provider-a", SurfaceId: "endpoint-a"},
		}},
		Capabilities: []Capability{capability},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	response, err := service.ProbeProvider(context.Background(), "provider-a", TriggerConnect)
	if err != nil {
		t.Fatalf("ProbeProvider() error = %v", err)
	}
	if got, want := len(capability.calls), 1; got != want {
		t.Fatalf("capability calls = %d, want %d", got, want)
	}
	call := capability.calls[0]
	if got, want := call.ProviderSurfaceBindingID, "endpoint-a"; got != want {
		t.Fatalf("target endpoint = %q, want %q", got, want)
	}
	if got, want := call.OwnerID, "codex"; got != want {
		t.Fatalf("target owner = %q, want %q", got, want)
	}
	if got, want := response.GetCliId(), "codex"; got != want {
		t.Fatalf("response cli_id = %q, want %q", got, want)
	}
	if got, want := response.GetOutcome(), managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_EXECUTED; got != want {
		t.Fatalf("response outcome = %v, want %v", got, want)
	}
}

func TestServiceProbeProviderReturnsUnsupportedWhenNoCapabilityMatches(t *testing.T) {
	service, err := NewService(Config{
		ProviderSurfaceBindings: fakeSurfaceBindingLister{items: []*managementv1.ProviderSurfaceBindingView{
			{ProviderId: "provider-a", SurfaceId: "endpoint-a"},
		}},
		Capabilities: []Capability{&fakeCapability{kind: OwnerKindVendor}},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	response, err := service.ProbeProvider(context.Background(), "provider-a", TriggerManual)
	if err != nil {
		t.Fatalf("ProbeProvider() error = %v", err)
	}
	if got, want := response.GetOutcome(), managementv1.ProviderOAuthObservabilityProbeOutcome_PROVIDER_O_AUTH_OBSERVABILITY_PROBE_OUTCOME_UNSUPPORTED; got != want {
		t.Fatalf("response outcome = %v, want %v", got, want)
	}
}

func TestServiceProbeProviderPropagatesListError(t *testing.T) {
	service, err := NewService(Config{
		ProviderSurfaceBindings: fakeSurfaceBindingLister{err: errors.New("boom")},
		Capabilities:            []Capability{&fakeCapability{kind: OwnerKindVendor}},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.ProbeProvider(context.Background(), "provider-a", TriggerManual); err == nil {
		t.Fatal("ProbeProvider() error is nil")
	}
}

type fakeSurfaceBindingLister struct {
	items []*managementv1.ProviderSurfaceBindingView
	err   error
}

func (l fakeSurfaceBindingLister) ListProviderSurfaceBindings(context.Context) ([]*managementv1.ProviderSurfaceBindingView, error) {
	return l.items, l.err
}

type fakeCapability struct {
	kind           OwnerKind
	ownerID        string
	matchSurfaceID string
	result         *ProbeResult
	calls          []ProbeTarget
}

func (c *fakeCapability) OwnerKind() OwnerKind {
	return c.kind
}

func (c *fakeCapability) Supports(surface *managementv1.ProviderSurfaceBindingView) (string, bool) {
	if c.matchSurfaceID != "" && surface.GetSurfaceId() != c.matchSurfaceID {
		return "", false
	}
	return c.ownerID, c.ownerID != ""
}

func (c *fakeCapability) ProbeProvider(_ context.Context, target ProbeTarget, _ Trigger) (*ProbeResult, error) {
	c.calls = append(c.calls, target)
	return c.result, nil
}
