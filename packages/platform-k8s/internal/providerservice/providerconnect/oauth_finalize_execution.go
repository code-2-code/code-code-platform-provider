package providerconnect

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
)

type providerConnectSessionFinalizer interface {
	Finalize(
		ctx context.Context,
		record *sessionRecord,
		oauthState *credentialv1.OAuthAuthorizationSessionState,
	) (*ProviderSurfaceBindingView, error)
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
) (*ProviderSurfaceBindingView, error) {
	if r.resources.providers == nil || r.queries == nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: oauth finalize runtime is incomplete")
	}
	plan, err := newOAuthFinalizePlan(record, oauthState)
	if err != nil {
		return nil, err
	}
	providerInput := plan.CreateProvider()
	provider, err := r.resources.providers.CreateProvider(ctx, providerInput)
	var surface *ProviderSurfaceBindingView
	if err != nil {
		if !isAlreadyExists(err) {
			return nil, err
		}
		existing, getErr := r.queries.FindSurface(ctx, plan.TargetSurfaceID())
		if getErr != nil {
			return nil, getErr
		}
		if err := plan.ValidateExisting(existing); err != nil {
			return nil, err
		}
		surface = existing
	} else {
		surface, err = findSurfaceBinding(provider, plan.TargetSurfaceID())
		if err != nil {
			return nil, err
		}
	}
	r.postConnect.Dispatch(ctx, surface.GetProviderId())
	return surface, nil
}

func findSurfaceBinding(provider *ProviderView, surfaceID string) (*ProviderSurfaceBindingView, error) {
	surfaceID = strings.TrimSpace(surfaceID)
	if provider == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: created provider is nil")
	}
	for _, surface := range provider.GetSurfaces() {
		if surface.GetSurfaceId() == surfaceID {
			return cloneProviderSurfaceBindingView(surface), nil
		}
	}
	return nil, domainerror.NewNotFound(
		"platformk8s/providerconnect: provider surface binding %q not found in created provider %q",
		surfaceID,
		provider.GetProviderId(),
	)
}
