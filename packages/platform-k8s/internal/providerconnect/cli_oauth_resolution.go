package providerconnect

import (
	"context"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

type cliOAuthResolvedTarget struct {
	target *connectTarget
	flow   credentialv1.OAuthAuthorizationFlow
}

func (r *cliOAuthResolvedTarget) StartSession(
	ctx context.Context,
	runtime providerConnectSessionStartRuntime,
) (*SessionView, error) {
	if r == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: cli oauth target is nil")
	}
	return newOAuthSessionStartExecution(r.target, r.flow).Execute(ctx, runtime)
}

type providerConnectCLIOAuthResolutionRuntime struct {
	support providerConnectSupport
	queries *providerConnectQueries
}

func newProviderConnectCLIOAuthResolutionRuntime(
	support providerConnectSupport,
	queries *providerConnectQueries,
) providerConnectCLIOAuthResolutionRuntime {
	return providerConnectCLIOAuthResolutionRuntime{
		support: support,
		queries: queries,
	}
}

func (r providerConnectCLIOAuthResolutionRuntime) ResolveConnect(
	ctx context.Context,
	command *ConnectCommand,
) (*cliOAuthResolvedTarget, error) {
	surfaceID := command.SurfaceID()
	if surfaceID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: surface_id is required for CLI OAuth")
	}
	surface, err := r.queries.LoadSurfaceMetadata(ctx, surfaceID)
	if err != nil {
		return nil, err
	}
	cliID := command.CLIID()
	if cliID == "" {
		cliID = strings.TrimSpace(surface.value.GetCli().GetCliId())
	}
	if cliID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface %q does not expose a CLI endpoint", surfaceID)
	}
	cli, err := r.loadCLISupport(ctx, cliID)
	if err != nil {
		return nil, err
	}
	flow, err := cli.Flow()
	if err != nil {
		return nil, err
	}
	displayName := cli.DisplayNameOr(command.DisplayName())
	candidate, err := newCLIOAuthCandidate(displayName, cliID, surfaceID)
	if err != nil {
		return nil, err
	}
	if err := surface.ValidateCandidate(candidate, credentialv1.CredentialKind_CREDENTIAL_KIND_OAUTH); err != nil {
		return nil, err
	}
	target, err := candidate.CLIOAuthTarget(displayName, cliID)
	if err != nil {
		return nil, err
	}
	return &cliOAuthResolvedTarget{target: target, flow: flow}, nil
}

func (r providerConnectCLIOAuthResolutionRuntime) ResolveReauthorize(
	ctx context.Context,
	provider *ProviderView,
) (*cliOAuthResolvedTarget, error) {
	target, err := newCLIReauthorizationTarget(provider)
	if err != nil {
		return nil, err
	}
	cli, err := r.loadCLISupport(ctx, target.CLIID)
	if err != nil {
		return nil, err
	}
	flow, err := cli.Flow()
	if err != nil {
		return nil, err
	}
	return &cliOAuthResolvedTarget{target: target, flow: flow}, nil
}

func (r providerConnectCLIOAuthResolutionRuntime) loadCLISupport(
	ctx context.Context,
	cliID string,
) (*cliOAuth, error) {
	if r.support.clis == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: cli support reader is nil")
	}
	cli, err := r.support.clis.Get(ctx, cliID)
	if err != nil {
		return nil, domainerror.NewNotFound("platformk8s/providerconnect: cli support %q not found", cliID)
	}
	return newCLIOAuth(cliID, cli), nil
}

type cliOAuth struct {
	cliID string
	value *supportv1.CLI
}

func newCLIOAuth(cliID string, value *supportv1.CLI) *cliOAuth {
	return &cliOAuth{
		cliID: strings.TrimSpace(cliID),
		value: value,
	}
}

func (p *cliOAuth) Flow() (credentialv1.OAuthAuthorizationFlow, error) {
	flow := credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_UNSPECIFIED
	if p != nil && p.value != nil && p.value.GetOauth() != nil {
		flow = p.value.GetOauth().GetFlow()
	}
	if flow == credentialv1.OAuthAuthorizationFlow_O_AUTH_AUTHORIZATION_FLOW_UNSPECIFIED {
		return flow, domainerror.NewValidation("platformk8s/providerconnect: cli %q does not expose oauth flow", p.CLIID())
	}
	return flow, nil
}

func (p *cliOAuth) DisplayNameOr(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	if p != nil && p.value != nil {
		if displayName := strings.TrimSpace(p.value.GetDisplayName()); displayName != "" {
			return displayName
		}
	}
	return p.CLIID()
}

func (p *cliOAuth) CLIID() string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(p.cliID)
}
