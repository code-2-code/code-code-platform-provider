package providerconnect

import (
	"context"
)

func (r providerConnectRuntime) connectWithAPIKey(ctx context.Context, command *ConnectCommand) (*ConnectResult, error) {
	if err := command.ValidateAPIKey(); err != nil {
		return nil, err
	}
	resolved, err := r.apiKeyResolutionRuntime().Resolve(ctx, command)
	if err != nil {
		return nil, err
	}
	result, err := resolved.Execute(ctx, command.CredentialID(), r.apiKeyConnectRuntime())
	if err != nil {
		return nil, err
	}
	return &ConnectResult{Provider: result.Provider}, nil
}

func (r providerConnectRuntime) apiKeyConnectRuntime() apiKeyConnectRuntime {
	return r.resources.APIKeyConnectRuntime()
}
