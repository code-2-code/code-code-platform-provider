package providerconnect

import (
	"context"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
)

func TestProviderConnectSessionQueryRuntimeGetSyncsPersistsAndProjectsView(t *testing.T) {
	store := newTestSessionStore(t)
	record, err := newSessionRecord(
		"session-query",
		testCLIOAuthSessionTarget(""),
		&credentialv1.OAuthAuthorizationSessionStatus{
			Phase: credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PENDING,
		},
	)
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	if err := store.create(context.Background(), record); err != nil {
		t.Fatalf("store.create() error = %v", err)
	}
	runtime := newProviderConnectSessionQueryRuntime(
		store,
		newProviderConnectSessionSyncRuntime(
			&oauthSessionServiceStub{
				getState: testOAuthSessionState(
					"session-query",
					credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED,
				),
			},
			&sessionFinalizerStub{},
		),
		newProviderConnectSessionViewRuntime(newProviderConnectQueries(
			nil,
			providerReaderStub{items: map[string]*ProviderView{
				"provider-codex": {
					ProviderID:  "provider-codex",
					DisplayName: "Codex Provider",
				},
			}},
			nil,
		)),
	)

	view, err := runtime.Get(context.Background(), "session-query")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := view.GetMessage(), "Provider authentication updated."; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
	if view.GetProvider() == nil {
		t.Fatal("provider = nil, want non-nil")
	}
	if got, want := view.GetProvider().GetProviderId(), "provider-codex"; got != want {
		t.Fatalf("provider_id = %q, want %q", got, want)
	}
	next, err := store.get(context.Background(), "session-query")
	if err != nil {
		t.Fatalf("store.get() error = %v", err)
	}
	if got, want := next.Message, "Provider authentication updated."; got != want {
		t.Fatalf("stored message = %q, want %q", got, want)
	}
}

type providerReaderStub struct {
	items map[string]*ProviderView
}

func (s providerReaderStub) Get(_ context.Context, providerID string) (*ProviderView, error) {
	return s.items[providerID], nil
}
