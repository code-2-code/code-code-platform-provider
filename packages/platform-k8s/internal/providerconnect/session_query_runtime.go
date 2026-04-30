package providerconnect

import (
	"context"
	"fmt"
)

type providerConnectSessionQueryRuntime struct {
	store providerConnectSessionStore
	sync  providerConnectSessionSyncRuntime
	views providerConnectSessionViewRuntime
}

func newProviderConnectSessionQueryRuntime(
	store providerConnectSessionStore,
	sync providerConnectSessionSyncRuntime,
	views providerConnectSessionViewRuntime,
) providerConnectSessionQueryRuntime {
	return providerConnectSessionQueryRuntime{
		store: store,
		sync:  sync,
		views: views,
	}
}

func (r providerConnectSessionQueryRuntime) Get(
	ctx context.Context,
	sessionID string,
) (*SessionView, error) {
	if r.store == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: session query runtime is incomplete")
	}
	record, err := r.store.get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	next, oauthState, err := r.sync.Sync(ctx, record)
	if err != nil {
		return nil, err
	}
	if err := r.store.put(ctx, next); err != nil {
		return nil, err
	}
	return r.views.View(ctx, next, oauthState)
}
