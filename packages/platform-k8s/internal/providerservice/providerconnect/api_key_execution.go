package providerconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

type apiKeyConnectExecution struct {
	credential       *CredentialAPIKeyCreate
	targetProviderID string
	targets          []*connectTarget
}

type apiKeyConnectRuntime struct {
	CreateCredential func(context.Context, CredentialAPIKeyCreate) (string, error)
	DeleteCredential func(context.Context, string) error
	CreateProvider   func(context.Context, *providerv1.Provider) (*ProviderView, error)
	Logger           *slog.Logger
}

type apiKeyConnectResult struct {
	TargetProviderID string
	Provider         *ProviderView
}

func newCustomAPIKeyConnectExecution(target *connectTarget, apiKey string) *apiKeyConnectExecution {
	if target == nil {
		return &apiKeyConnectExecution{}
	}
	return &apiKeyConnectExecution{
		credential:       target.APIKeyCredential(apiKey),
		targetProviderID: target.TargetProviderID,
		targets:          []*connectTarget{target},
	}
}

func newVendorAPIKeyConnectExecution(plan *connectPlan, apiKey string) *apiKeyConnectExecution {
	if plan == nil {
		return &apiKeyConnectExecution{}
	}
	return &apiKeyConnectExecution{
		credential:       plan.APIKeyCredential(apiKey),
		targetProviderID: plan.TargetProviderID,
		targets:          append([]*connectTarget(nil), plan.Targets...),
	}
}

func (e *apiKeyConnectExecution) Execute(ctx context.Context, runtime apiKeyConnectRuntime) (*apiKeyConnectResult, error) {
	if err := e.validate(); err != nil {
		return nil, err
	}
	credentialID, err := runtime.CreateCredential(ctx, *e.credential)
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: create credential: %w", err)
	}
	providerInput := e.provider(credentialID)
	provider, createErr := runtime.CreateProvider(ctx, providerInput)
	if createErr != nil {
		if rollbackErr := e.rollback(ctx, runtime, credentialID); rollbackErr != nil {
			return nil, fmt.Errorf("platformk8s/providerconnect: create provider: %w", errors.Join(createErr, rollbackErr))
		}
		return nil, fmt.Errorf("platformk8s/providerconnect: create provider: %w", createErr)
	}
	if provider == nil {
		err := domainerror.NewValidation("platformk8s/providerconnect: created provider is nil")
		if rollbackErr := e.rollback(ctx, runtime, credentialID); rollbackErr != nil {
			return nil, errors.Join(err, rollbackErr)
		}
		return nil, err
	}
	return &apiKeyConnectResult{
		TargetProviderID: e.targetProviderID,
		Provider:         provider,
	}, nil
}

func (e *apiKeyConnectExecution) validate() error {
	if e == nil || e.credential == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: api key credential request is required")
	}
	if len(e.targets) == 0 {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface target is required")
	}
	return nil
}

func (e *apiKeyConnectExecution) provider(credentialID string) *providerv1.Provider {
	if len(e.targets) == 1 {
		return e.targets[0].Provider(credentialID)
	}
	base := e.targets[0].Provider(credentialID)
	base.Surfaces = make([]*providerv1.ProviderSurfaceBinding, 0, len(e.targets))
	for _, target := range e.targets {
		if target == nil {
			continue
		}
		surface := target.ProviderSurfaceBinding("")
		if credentialID != "" {
			surface.ProviderCredentialRef = &providerv1.ProviderCredentialRef{ProviderCredentialId: credentialID}
		}
		base.Surfaces = append(base.Surfaces, surface)
	}
	return base
}

func (e *apiKeyConnectExecution) rollback(
	ctx context.Context,
	runtime apiKeyConnectRuntime,
	credentialID string,
) error {
	rollbackCtx := context.WithoutCancel(ctx)
	if runtime.DeleteCredential == nil || credentialID == "" {
		return nil
	}
	if err := runtime.DeleteCredential(rollbackCtx, credentialID); err != nil {
		logRollbackFailure(runtime.Logger, "credential", credentialID, err)
		return fmt.Errorf("rollback credential %q: %w", credentialID, err)
	}
	return nil
}

func logRollbackFailure(logger *slog.Logger, kind, resourceID string, err error) {
	if logger == nil || err == nil {
		return
	}
	if kind == "credential" {
		logger.Warn("platformk8s/providerconnect: rollback credential failed", "credential_id", resourceID, "error", err)
	}
}
