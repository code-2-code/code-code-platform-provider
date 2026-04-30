package providerconnect

import (
	"context"
	"fmt"
	"log/slog"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
)

type providerConnectSessionFinalizer interface {
	Finalize(
		ctx context.Context,
		record *sessionRecord,
		oauthState *credentialv1.OAuthAuthorizationSessionState,
	) (*ProviderView, error)
}

type providerConnectOAuthFinalizeRuntime struct {
	resources   providerConnectResources
	queries     *providerConnectQueries
	postConnect *providerConnectPostConnectWorkflow
	logger      *slog.Logger
}

func newProviderConnectOAuthFinalizeRuntime(
	resources providerConnectResources,
	queries *providerConnectQueries,
	postConnect *providerConnectPostConnectWorkflow,
	logger *slog.Logger,
) providerConnectOAuthFinalizeRuntime {
	if logger == nil {
		logger = slog.Default()
	}
	return providerConnectOAuthFinalizeRuntime{
		resources:   resources,
		queries:     queries,
		postConnect: postConnect,
		logger:      logger,
	}
}

func (r providerConnectOAuthFinalizeRuntime) Finalize(
	ctx context.Context,
	record *sessionRecord,
	oauthState *credentialv1.OAuthAuthorizationSessionState,
) (*ProviderView, error) {
	if r.resources.providers == nil || r.queries == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: oauth finalize runtime is incomplete")
	}
	plan, err := newOAuthFinalizePlan(record, oauthState)
	if err != nil {
		return nil, err
	}
	providerInput := plan.CreateProvider()
	provider, err := r.resources.providers.CreateProvider(ctx, providerInput)
	var finalProvider *ProviderView
	if err != nil {
		if !isAlreadyExists(err) {
			return nil, err
		}
		existing, getErr := r.queries.FindProviderBySurface(ctx, plan.TargetSurfaceID())
		if getErr != nil {
			return nil, getErr
		}
		if err := plan.ValidateExisting(existing); err != nil {
			return nil, err
		}
		finalProvider = existing
	} else {
		if provider == nil {
			return nil, domainerror.NewValidation("platformk8s/providerconnect: created provider is nil")
		}
		if provider.GetSurfaceId() != plan.TargetSurfaceID() {
			return nil, domainerror.NewNotFound(
				"platformk8s/providerconnect: provider surface %q not found in created provider %q",
				plan.TargetSurfaceID(),
				provider.GetProviderId(),
			)
		}
		finalProvider = provider
	}
	r.postConnect.Dispatch(ctx, finalProvider.GetProviderId())
	return finalProvider, nil
}
