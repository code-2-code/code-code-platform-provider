package providerconnect

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
)

func newCLIReauthorizationTarget(provider *ProviderView) (*connectTarget, error) {
	if provider == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider is nil")
	}
	if len(provider.GetSurfaces()) == 0 {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider %q has no surfaces", provider.GetProviderId())
	}
	surface := provider.GetSurfaces()[0]
	runtime := surface.GetRuntime()
	if runtime == nil || runtime.GetCli() == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider %q is not backed by CLI OAuth", provider.GetProviderId())
	}
	if strings.TrimSpace(provider.GetProviderCredentialId()) == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider %q credential is missing", provider.GetProviderId())
	}
	cliID := strings.TrimSpace(runtime.GetCli().GetCliId())
	if cliID == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider %q cli_id is missing", provider.GetProviderId())
	}
	return newConnectTargetWithIDs(
		AddMethodCLIOAuth,
		provider.GetDisplayName(),
		provider.GetVendorId(),
		cliID,
		surface.GetSurfaceId(),
		provider.GetProviderCredentialId(),
		provider.GetProviderId(),
		runtime,
	), nil
}
