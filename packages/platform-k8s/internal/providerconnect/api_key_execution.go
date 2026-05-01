package providerconnect

import (
	"context"
	"fmt"
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

type apiKeyConnectExecution struct {
	credentialID     string
	targetProviderID string
	targets          []*connectTarget
}

type apiKeyConnectRuntime struct {
	CreateProvider func(context.Context, *providerv1.Provider) (*ProviderView, error)
}

type apiKeyConnectResult struct {
	TargetProviderID string
	Provider         *ProviderView
}

func newCustomAPIKeyConnectExecution(target *connectTarget, credentialID string) *apiKeyConnectExecution {
	if target == nil {
		return &apiKeyConnectExecution{}
	}
	return &apiKeyConnectExecution{
		credentialID:     strings.TrimSpace(credentialID),
		targetProviderID: target.TargetProviderID,
		targets:          []*connectTarget{target},
	}
}

func (e *apiKeyConnectExecution) Execute(ctx context.Context, runtime apiKeyConnectRuntime) (*apiKeyConnectResult, error) {
	if err := e.validate(); err != nil {
		return nil, err
	}
	providerInput := e.provider(e.credentialID)
	provider, err := runtime.CreateProvider(ctx, providerInput)
	if err != nil {
		return nil, fmt.Errorf("platformk8s/providerconnect: create provider: %w", err)
	}
	if provider == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: created provider is nil")
	}
	return &apiKeyConnectResult{
		TargetProviderID: e.targetProviderID,
		Provider:         provider,
	}, nil
}

func (e *apiKeyConnectExecution) validate() error {
	if e == nil || strings.TrimSpace(e.credentialID) == "" {
		return domainerror.NewValidation("platformk8s/providerconnect: credential_id is required")
	}
	if len(e.targets) == 0 {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface target is required")
	}
	return nil
}

func (e *apiKeyConnectExecution) provider(credentialID string) *providerv1.Provider {
	return e.targets[0].Provider(credentialID)
}
