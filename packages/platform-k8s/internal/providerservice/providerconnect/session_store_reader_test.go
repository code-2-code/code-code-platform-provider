package providerconnect

import (
	"context"
	"fmt"
	"testing"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/platform-k8s/internal/platform/testutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type sessionStoreGetNotFoundClient struct {
	ctrlclient.Client
}

func (c sessionStoreGetNotFoundClient) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	return apierrors.NewNotFound(
		schema.GroupResource{Resource: "configmaps"},
		key.Name,
	)
}

type sessionStoreConflictOnceClient struct {
	ctrlclient.Client
	conflicted bool
}

func (c *sessionStoreConflictOnceClient) Update(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.UpdateOption) error {
	if !c.conflicted {
		c.conflicted = true
		return apierrors.NewConflict(
			schema.GroupResource{Resource: "configmaps"},
			obj.GetName(),
			fmt.Errorf("stale resource version"),
		)
	}
	return c.Client.Update(ctx, obj, opts...)
}

func TestSessionStoreUsesReaderForReadAndUpdate(t *testing.T) {
	reader := ctrlclientfake.NewClientBuilder().
		WithScheme(testutil.NewScheme()).
		Build()
	store, err := newSessionStore(sessionStoreGetNotFoundClient{Client: reader}, reader, "code-code")
	if err != nil {
		t.Fatalf("newSessionStore() error = %v", err)
	}
	record, err := newSessionRecord(
		"session-reader",
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

	stored, err := store.get(context.Background(), "session-reader")
	if err != nil {
		t.Fatalf("store.get() error = %v", err)
	}
	if got, want := stored.OAuthSessionID, "session-reader"; got != want {
		t.Fatalf("oauth_session_id = %q, want %q", got, want)
	}

	record.Message = "Provider authentication updated."
	if err := store.put(context.Background(), record); err != nil {
		t.Fatalf("store.put() error = %v", err)
	}
	next, err := store.get(context.Background(), "session-reader")
	if err != nil {
		t.Fatalf("store.get() after put error = %v", err)
	}
	if got, want := next.Message, "Provider authentication updated."; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestSessionStorePutRetriesConflict(t *testing.T) {
	base := ctrlclientfake.NewClientBuilder().
		WithScheme(testutil.NewScheme()).
		Build()
	client := &sessionStoreConflictOnceClient{Client: base}
	store, err := newSessionStore(client, base, "code-code")
	if err != nil {
		t.Fatalf("newSessionStore() error = %v", err)
	}
	record, err := newSessionRecord(
		"session-conflict",
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

	record.Message = "Retried provider authentication update."
	if err := store.put(context.Background(), record); err != nil {
		t.Fatalf("store.put() error = %v", err)
	}
	if !client.conflicted {
		t.Fatal("store.put() did not exercise conflict path")
	}
	next, err := store.get(context.Background(), "session-conflict")
	if err != nil {
		t.Fatalf("store.get() after put error = %v", err)
	}
	if got, want := next.Message, "Retried provider authentication update."; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}
