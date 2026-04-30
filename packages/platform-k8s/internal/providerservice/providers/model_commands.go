package providers

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
)

func (p *ProviderProjection) ValidateMutable() error {
	if p == nil || p.value == nil {
		return domainerror.NewValidation("platformk8s/providers: provider is nil")
	}
	if p.ID() == "" {
		return domainerror.NewValidation("platformk8s/providers: provider id is empty")
	}
	if p.value.GetSurfaceId() == "" {
		return domainerror.NewValidation("platformk8s/providers: provider %q has no surface", p.ID())
	}
	return nil
}

func (p *ProviderProjection) Rename(displayName string) (string, error) {
	if err := p.ValidateMutable(); err != nil {
		return "", err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return "", domainerror.NewValidation("platformk8s/providers: display name is required")
	}
	return displayName, nil
}
