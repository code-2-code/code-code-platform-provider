package providerconnect

import (
	"context"
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

type apiKeyResolvedConnect struct {
	target *connectTarget
	plan   *connectPlan
}

func newCustomAPIKeyResolvedConnect(target *connectTarget) *apiKeyResolvedConnect {
	return &apiKeyResolvedConnect{target: target}
}

func newVendorAPIKeyResolvedConnect(plan *connectPlan) *apiKeyResolvedConnect {
	return &apiKeyResolvedConnect{plan: plan}
}

func (r *apiKeyResolvedConnect) Execute(
	ctx context.Context,
	apiKey string,
	runtime apiKeyConnectRuntime,
) (*apiKeyConnectResult, error) {
	switch {
	case r == nil:
		return nil, domainerror.NewValidation("platformk8s/providerconnect: api key connect target is required")
	case r.plan != nil:
		return newVendorAPIKeyConnectExecution(r.plan, apiKey).Execute(ctx, runtime)
	case r.target != nil:
		return newCustomAPIKeyConnectExecution(r.target, apiKey).Execute(ctx, runtime)
	default:
		return nil, domainerror.NewValidation("platformk8s/providerconnect: api key connect target is required")
	}
}

type providerConnectAPIKeyResolutionRuntime struct {
	support providerConnectSupport
	queries *providerConnectQueries
}

func newProviderConnectAPIKeyResolutionRuntime(
	support providerConnectSupport,
	queries *providerConnectQueries,
) providerConnectAPIKeyResolutionRuntime {
	return providerConnectAPIKeyResolutionRuntime{
		support: support,
		queries: queries,
	}
}

func (r providerConnectAPIKeyResolutionRuntime) Resolve(
	ctx context.Context,
	command *ConnectCommand,
) (*apiKeyResolvedConnect, error) {
	if command.IsVendorAPIKey() {
		return r.ResolveVendor(ctx, command)
	}
	return r.ResolveCustom(ctx, command)
}

func (r providerConnectAPIKeyResolutionRuntime) ResolveCustom(
	ctx context.Context,
	command *ConnectCommand,
) (*apiKeyResolvedConnect, error) {
	displayName := command.DisplayNameOr("Custom API Key")
	catalogs, err := newSurfaceCatalogSet(command.SurfaceModelCatalogs())
	if err != nil {
		return nil, err
	}
	candidate, err := newCustomAPIKeyCandidate(displayName, command.APIKeyInput(), catalogs)
	if err != nil {
		return nil, err
	}
	definition, err := r.queries.LoadSurfaceMetadata(ctx, candidate.SurfaceID())
	if err != nil {
		return nil, err
	}
	if err := definition.ValidateCandidate(candidate, credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY); err != nil {
		return nil, err
	}
	target, err := candidate.CustomAPIKeyTarget(displayName)
	if err != nil {
		return nil, err
	}
	return newCustomAPIKeyResolvedConnect(target), nil
}

func (r providerConnectAPIKeyResolutionRuntime) ResolveVendor(
	ctx context.Context,
	command *ConnectCommand,
) (*apiKeyResolvedConnect, error) {
	vendorID := command.VendorID()
	if vendorID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: vendor_id is required for vendor API key connect")
	}
	vendor, err := r.loadVendorSupport(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	displayName := vendor.DisplayNameOr(command.DisplayName())
	catalogs, err := newSurfaceCatalogSet(command.SurfaceModelCatalogs())
	if err != nil {
		return nil, err
	}
	candidates, err := vendor.Candidates(catalogs)
	if err != nil {
		return nil, err
	}
	targets := make([]*connectTarget, 0, len(candidates))
	for _, candidate := range candidates {
		definition, err := r.queries.LoadSurfaceMetadata(ctx, candidate.SurfaceID())
		if err != nil {
			return nil, err
		}
		if err := definition.ValidateCandidate(candidate, credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY); err != nil {
			return nil, err
		}
		target, err := candidate.VendorAPIKeyTarget(displayName, vendor.VendorID())
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	plan, err := newConnectPlan(displayName, vendor.VendorID(), targets)
	if err != nil {
		return nil, err
	}
	return newVendorAPIKeyResolvedConnect(plan), nil
}

func (r providerConnectAPIKeyResolutionRuntime) loadVendorSupport(
	ctx context.Context,
	vendorID string,
) (*vendorAPIKeySupport, error) {
	if r.support.vendors == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: vendor support reader is nil")
	}
	vendor, err := r.support.vendors.GetForConnect(ctx, vendorID)
	if err != nil {
		return nil, domainerror.NewNotFound("platformk8s/providerconnect: vendor support %q not found", vendorID)
	}
	return newVendorAPIKeySupport(vendorID, vendor), nil
}

type vendorAPIKeySupport struct {
	vendorID string
	value    *supportv1.Vendor
}

func newVendorAPIKeySupport(vendorID string, value *supportv1.Vendor) *vendorAPIKeySupport {
	return &vendorAPIKeySupport{
		vendorID: strings.TrimSpace(vendorID),
		value:    value,
	}
}

func (p *vendorAPIKeySupport) DisplayNameOr(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	if p != nil && p.value != nil && p.value.GetVendor() != nil {
		if displayName := strings.TrimSpace(p.value.GetVendor().GetDisplayName()); displayName != "" {
			return displayName
		}
	}
	return p.VendorID()
}

func (p *vendorAPIKeySupport) VendorID() string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(p.vendorID)
}

func (p *vendorAPIKeySupport) Candidates(
	catalogs *surfaceCatalogSet,
) ([]*connectSurfaceBindingCandidate, error) {
	return newVendorAPIKeyCandidates(p.value, catalogs)
}
