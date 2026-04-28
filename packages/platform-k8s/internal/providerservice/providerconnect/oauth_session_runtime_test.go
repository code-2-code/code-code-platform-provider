package providerconnect

import (
	"context"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/platform-k8s/internal/platform/testutil"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOAuthSessionStartExecutionCreatesRecordAndView(t *testing.T) {
	store := newTestSessionStore(t)
	oauth := &oauthSessionServiceStub{
		startState: testOAuthSessionState(
			"session-start",
			credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_AWAITING_USER,
		),
	}
	runtime := newProviderConnectSessionStartRuntime(
		newProviderConnectSessions(oauth, store),
		newProviderConnectSessionViewRuntime(nil),
	)

	view, err := newOAuthSessionStartExecution(
		testCLIOAuthSessionTarget("codex"),
		credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_DEVICE,
	).Execute(context.Background(), runtime)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := view.GetSessionId(), "session-start"; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if got, want := view.GetPhase(), SessionPhaseAwaitingUser; got != want {
		t.Fatalf("phase = %v, want %v", got, want)
	}
	record, err := store.get(context.Background(), "session-start")
	if err != nil {
		t.Fatalf("store.get() error = %v", err)
	}
	if got, want := record.ProviderSurfaceID, "codex"; got != want {
		t.Fatalf("provider_surface_id = %q, want %q", got, want)
	}
	if got := oauth.cancelCalls; got != 0 {
		t.Fatalf("cancel_calls = %d, want 0", got)
	}
}

func TestOAuthSessionStartExecutionCancelsSessionWhenStoreCreateFails(t *testing.T) {
	store := newTestSessionStore(t)
	oauth := &oauthSessionServiceStub{
		startState: testOAuthSessionState(
			"session-duplicate",
			credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PENDING,
		),
	}
	runtime := newProviderConnectSessionStartRuntime(
		newProviderConnectSessions(oauth, store),
		newProviderConnectSessionViewRuntime(nil),
	)
	record, err := newSessionRecord("session-duplicate", testCLIOAuthSessionTarget("codex"), oauth.startState.GetStatus())
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	if err := store.create(context.Background(), record); err != nil {
		t.Fatalf("store.create() error = %v", err)
	}

	_, err = newOAuthSessionStartExecution(
		testCLIOAuthSessionTarget("codex"),
		credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_DEVICE,
	).Execute(context.Background(), runtime)
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if got, want := oauth.cancelCalls, 1; got != want {
		t.Fatalf("cancel_calls = %d, want %d", got, want)
	}
	if got, want := oauth.canceledSessionIDs[0], "session-duplicate"; got != want {
		t.Fatalf("canceled_session_id = %q, want %q", got, want)
	}
}

func TestProviderConnectSessionSyncRuntimeFinalizesPendingInstance(t *testing.T) {
	record, err := newSessionRecord(
		"session-sync",
		testCLIOAuthSessionTarget("codex"),
		&credentialv1.OAuthAuthorizationSessionStatus{
			Phase: credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PENDING,
		},
	)
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	finalizer := &sessionFinalizerStub{
		instance: &ProviderSurfaceBindingView{SurfaceID: "codex"},
	}
	state := testOAuthSessionState(
		"session-sync",
		credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED,
	)
	state.Status.ImportedCredential = &credentialv1.ImportedCredentialSummary{CredentialId: "credential-codex"}
	runtime := newProviderConnectSessionSyncRuntime(
		&oauthSessionServiceStub{getState: state},
		finalizer,
	)

	next, oauthState, err := runtime.Sync(context.Background(), record)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if oauthState == nil {
		t.Fatal("oauthState = nil, want non-nil")
	}
	if got, want := finalizer.calls, 1; got != want {
		t.Fatalf("finalizer.calls = %d, want %d", got, want)
	}
	if got, want := next.ConnectedSurfaceID, "codex"; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got, want := next.Message, "Provider connected."; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestProviderConnectSessionSyncRuntimeDoesNotFinalizeProcessingSession(t *testing.T) {
	record, err := newSessionRecord(
		"session-sync",
		testCLIOAuthSessionTarget("codex"),
		&credentialv1.OAuthAuthorizationSessionStatus{
			Phase: credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_AWAITING_USER,
		},
	)
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	finalizer := &sessionFinalizerStub{
		instance: &ProviderSurfaceBindingView{SurfaceID: "codex"},
	}
	state := testOAuthSessionState(
		"session-sync",
		credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PROCESSING,
	)
	state.Spec.TargetCredentialId = "credential-codex"
	runtime := newProviderConnectSessionSyncRuntime(
		&oauthSessionServiceStub{getState: state},
		finalizer,
	)

	next, _, err := runtime.Sync(context.Background(), record)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got, want := finalizer.calls, 0; got != want {
		t.Fatalf("finalizer.calls = %d, want %d", got, want)
	}
	if got, want := next.ConnectedSurfaceID, ""; got != want {
		t.Fatalf("surface_id = %q, want %q", got, want)
	}
	if got, want := next.Phase, SessionPhaseProcessing; got != want {
		t.Fatalf("phase = %v, want %v", got, want)
	}
}

func TestProviderConnectSessionSyncRuntimeMarksAuthenticationUpdatedWithoutFinalize(t *testing.T) {
	record, err := newSessionRecord(
		"session-reauth",
		testCLIOAuthSessionTarget(""),
		&credentialv1.OAuthAuthorizationSessionStatus{
			Phase: credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_PENDING,
		},
	)
	if err != nil {
		t.Fatalf("newSessionRecord() error = %v", err)
	}
	finalizer := &sessionFinalizerStub{}
	runtime := newProviderConnectSessionSyncRuntime(
		&oauthSessionServiceStub{
			getState: testOAuthSessionState(
				"session-reauth",
				credentialv1.OAuthAuthorizationPhase_O_AUTH_AUTHORIZATION_PHASE_SUCCEEDED,
			),
		},
		finalizer,
	)

	next, _, err := runtime.Sync(context.Background(), record)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := finalizer.calls; got != 0 {
		t.Fatalf("finalizer.calls = %d, want 0", got)
	}
	if got, want := next.Message, "Provider authentication updated."; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func newTestSessionStore(t *testing.T) *sessionStore {
	t.Helper()
	client := ctrlclientfake.NewClientBuilder().
		WithScheme(testutil.NewScheme()).
		Build()
	store, err := newSessionStore(client, client, "code-code")
	if err != nil {
		t.Fatalf("newSessionStore() error = %v", err)
	}
	return store
}

func testCLIOAuthSessionTarget(surfaceID string) *connectTarget {
	runtime := testCLISurfaceRuntime("codex", "codex")
	return newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		"Codex",
		"openai",
		"codex",
		surfaceID,
		"credential-codex",
		"provider-codex",
		runtime,
	)
}

func testOAuthSessionState(
	sessionID string,
	phase credentialv1.OAuthAuthorizationPhase,
) *credentialv1.OAuthAuthorizationSessionState {
	return &credentialv1.OAuthAuthorizationSessionState{
		Spec: &credentialv1.OAuthAuthorizationSessionSpec{
			SessionId: sessionID,
		},
		Status: &credentialv1.OAuthAuthorizationSessionStatus{
			Phase:            phase,
			AuthorizationUrl: "https://auth.example.com/device",
			UserCode:         "ABCD-EFGH",
			Message:          "Authorize this device",
		},
	}
}

type oauthSessionServiceStub struct {
	startState         *credentialv1.OAuthAuthorizationSessionState
	getState           *credentialv1.OAuthAuthorizationSessionState
	startCalls         int
	getCalls           int
	cancelCalls        int
	canceledSessionIDs []string
}

func (s *oauthSessionServiceStub) StartSession(
	_ context.Context,
	_ *credentialv1.OAuthAuthorizationSessionSpec,
) (*credentialv1.OAuthAuthorizationSessionState, error) {
	s.startCalls += 1
	return s.startState, nil
}

func (s *oauthSessionServiceStub) GetSession(
	_ context.Context,
	_ string,
) (*credentialv1.OAuthAuthorizationSessionState, error) {
	s.getCalls += 1
	return s.getState, nil
}

func (s *oauthSessionServiceStub) CancelSession(
	_ context.Context,
	sessionID string,
) (*credentialv1.OAuthAuthorizationSessionState, error) {
	s.cancelCalls += 1
	s.canceledSessionIDs = append(s.canceledSessionIDs, sessionID)
	return s.startState, nil
}

type sessionFinalizerStub struct {
	instance *ProviderSurfaceBindingView
	calls    int
}

func (s *sessionFinalizerStub) Finalize(
	_ context.Context,
	_ *sessionRecord,
	_ *credentialv1.OAuthAuthorizationSessionState,
) (*ProviderSurfaceBindingView, error) {
	s.calls += 1
	return s.instance, nil
}
