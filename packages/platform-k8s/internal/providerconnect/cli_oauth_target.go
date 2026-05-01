package providerconnect

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
)

func newCLIReauthorizationTarget(provider *ProviderView) (*connectTarget, error) {
	if provider == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider is nil")
	}
	if strings.TrimSpace(provider.GetProviderCredentialId()) == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider %q credential is missing", provider.GetProviderId())
	}
	cliID := ""
	for _, endpoint := range provider.GetEndpoints() {
		if endpoint.GetCli() != nil {
			cliID = strings.TrimSpace(endpoint.GetCli().GetCliId())
			break
		}
	}
	if cliID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider %q cli_id is missing", provider.GetProviderId())
	}
	return newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		provider.GetDisplayName(),
		cliID,
		provider.GetSurfaceId(),
		provider.GetProviderCredentialId(),
		provider.GetProviderId(),
		provider.GetModels(),
	), nil
}
