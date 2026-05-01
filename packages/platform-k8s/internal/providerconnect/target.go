package providerconnect

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/resourcemeta"
)

type connectTarget struct {
	AddMethod           AddMethod
	DisplayName         string
	CLIID               string
	SurfaceID           string
	TargetCredentialID  string
	TargetProviderID    string
	Models              []*providerv1.ProviderModel
	CustomAPIKeySurface *providerv1.CustomAPIKeySurface
}

func newConnectTarget(
	addMethod AddMethod,
	displayName, cliID, surfaceID string,
	models []*providerv1.ProviderModel,
	suffix string,
) (*connectTarget, error) {
	targetCredentialID, err := resourcemeta.EnsureResourceID("", displayName, suffix)
	if err != nil {
		return nil, err
	}
	targetProviderID, err := resourcemeta.EnsureResourceID("", displayName, suffix+"-provider")
	if err != nil {
		return nil, err
	}
	return newConnectTargetWithIDs(
		addMethod,
		displayName,
		cliID,
		surfaceID,
		targetCredentialID,
		targetProviderID,
		models,
	), nil
}

func newConnectTargetWithIDs(
	addMethod AddMethod,
	displayName, cliID, surfaceID, targetCredentialID, targetProviderID string,
	models []*providerv1.ProviderModel,
) *connectTarget {
	return &connectTarget{
		AddMethod:          addMethod,
		DisplayName:        strings.TrimSpace(displayName),
		CLIID:              strings.TrimSpace(cliID),
		SurfaceID:          strings.TrimSpace(surfaceID),
		TargetCredentialID: strings.TrimSpace(targetCredentialID),
		TargetProviderID:   strings.TrimSpace(targetProviderID),
		Models:             cloneProviderModels(models),
	}
}

func (t *connectTarget) WithSharedIdentity(targetCredentialID, targetProviderID string) *connectTarget {
	if t == nil {
		return &connectTarget{}
	}
	next := newConnectTargetWithIDs(
		t.AddMethod,
		t.DisplayName,
		t.CLIID,
		t.SurfaceID,
		targetCredentialID,
		targetProviderID,
		t.Models,
	)
	next.CustomAPIKeySurface = cloneCustomAPIKeySurface(t.CustomAPIKeySurface)
	return next
}

func (t *connectTarget) OAuthSessionSpec(flow credentialv1.OAuthAuthorizationFlow) *credentialv1.OAuthAuthorizationSessionSpec {
	if t == nil {
		return &credentialv1.OAuthAuthorizationSessionSpec{}
	}
	return &credentialv1.OAuthAuthorizationSessionSpec{
		CliId:              strings.TrimSpace(t.CLIID),
		Flow:               flow,
		TargetCredentialId: strings.TrimSpace(t.TargetCredentialID),
		TargetDisplayName:  strings.TrimSpace(t.DisplayName),
	}
}

func (t *connectTarget) Provider(credentialID string) *providerv1.Provider {
	provider := &providerv1.Provider{
		ProviderId:  strings.TrimSpace(t.TargetProviderID),
		DisplayName: strings.TrimSpace(t.DisplayName),
		SurfaceId:   strings.TrimSpace(t.SurfaceID),
		Models:      cloneProviderModels(t.Models),
	}
	if cid := strings.TrimSpace(credentialID); cid != "" {
		provider.ProviderCredentialRef = &providerv1.ProviderCredentialRef{ProviderCredentialId: cid}
	}
	if t.CustomAPIKeySurface != nil {
		provider.CustomApiKeySurface = cloneCustomAPIKeySurface(t.CustomAPIKeySurface)
	}
	return provider
}

func cloneCustomAPIKeySurface(surface *providerv1.CustomAPIKeySurface) *providerv1.CustomAPIKeySurface {
	if surface == nil {
		return nil
	}
	return &providerv1.CustomAPIKeySurface{
		BaseUrl:  strings.TrimSpace(surface.GetBaseUrl()),
		Protocol: surface.GetProtocol(),
	}
}
