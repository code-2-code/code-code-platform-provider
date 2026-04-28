package providerconnect

import (
	"context"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
)

type providerConnectSessionViewRuntime struct {
	queries *providerConnectQueries
}

func newProviderConnectSessionViewRuntime(queries *providerConnectQueries) providerConnectSessionViewRuntime {
	return providerConnectSessionViewRuntime{queries: queries}
}

func (r providerConnectSessionViewRuntime) View(
	ctx context.Context,
	record *sessionRecord,
	oauthState *credentialv1.OAuthAuthorizationSessionState,
) (*SessionView, error) {
	if record == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: session record is nil")
	}
	var provider *ProviderView
	if r.queries != nil && record.TargetProviderID != "" {
		next, err := r.queries.FindProvider(ctx, record.TargetProviderID)
		if err != nil {
			if !isNotFound(err) {
				return nil, err
			}
		} else {
			provider = next
		}
	}
	return record.view(provider, oauthState), nil
}
