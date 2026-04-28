package providers

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

func (p *ProviderProjection) ValidateMutable() error {
	if p == nil || p.value == nil {
		return domainerror.NewValidation("platformk8s/providers: provider is nil")
	}
	if p.ID() == "" {
		return domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	if len(p.value.GetSurfaces()) == 0 {
		return domainerror.NewValidation("platformk8s/providers: provider %q has no surfaces", p.ID())
	}
	return nil
}

func (p *ProviderProjection) Rename(displayName string) (string, bool, error) {
	if err := p.ValidateMutable(); err != nil {
		return "", false, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return "", false, domainerror.NewValidation("platformk8s/providers: display name is required")
	}
	return displayName, p.CredentialID() != "", nil
}

func (p *ProviderProjection) APIKeyAuthenticationCredential(apiKey string) (*CredentialAPIKeyUpdate, error) {
	if err := p.ValidateMutable(); err != nil {
		return nil, err
	}
	if p.AuthKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
		return nil, domainerror.NewValidation("platformk8s/providers: provider %q does not accept API key authentication update", p.ID())
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: api key is required")
	}
	if p.CredentialID() == "" {
		return nil, domainerror.NewValidation("platformk8s/providers: provider %q does not reference a credential", p.ID())
	}
	return &CredentialAPIKeyUpdate{
		CredentialID: p.CredentialID(),
		DisplayName:  p.DisplayName(),
		Purpose:      credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_DATA_PLANE,
		VendorID:     p.VendorID(),
		APIKey:       apiKey,
	}, nil
}

func (p *ProviderProjection) ValidateCLIOAuthAuthentication() error {
	if err := p.ValidateMutable(); err != nil {
		return err
	}
	if p.AuthKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI {
		return domainerror.NewValidation("platformk8s/providers: provider %q does not accept CLI OAuth authentication update", p.ID())
	}
	return nil
}
