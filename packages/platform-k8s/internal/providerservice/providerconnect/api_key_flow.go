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
	result, err := resolved.Execute(ctx, command.APIKeyValue(), r.apiKeyConnectRuntime())
	if err != nil {
		return nil, err
	}
	r.postConnect.Dispatch(ctx, result.TargetProviderID)
	return &ConnectResult{Provider: result.Provider}, nil
}

func (r providerConnectRuntime) apiKeyConnectRuntime() apiKeyConnectRuntime {
	return r.resources.APIKeyConnectRuntime(r.logger)
}
