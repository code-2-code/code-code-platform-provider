package providerconnect

import (
	"context"
)

func (r providerConnectRuntime) connectWithCLIOAuth(ctx context.Context, command *ConnectCommand) (*ConnectResult, error) {
	if err := command.ValidateCLIOAuth(); err != nil {
		return nil, err
	}
	resolved, err := r.cliOAuthResolutionRuntime().ResolveConnect(ctx, command)
	if err != nil {
		return nil, err
	}
	view, err := resolved.StartSession(ctx, r.sessionStartRuntime())
	if err != nil {
		return nil, err
	}
	return &ConnectResult{Session: view}, nil
}

func (r providerConnectRuntime) Reauthorize(ctx context.Context, provider *ProviderView) (*SessionView, error) {
	resolved, err := r.cliOAuthResolutionRuntime().ResolveReauthorize(ctx, provider)
	if err != nil {
		return nil, err
	}
	return resolved.StartSession(ctx, r.sessionStartRuntime())
}
