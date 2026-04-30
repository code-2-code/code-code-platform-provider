package providerconnect

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/resourcemeta"
)

type connectTarget struct {
	AddMethod          AddMethod
	DisplayName        string
	VendorID           string
	CLIID              string
	SurfaceID          string
	TargetCredentialID string
	TargetProviderID   string
	RuntimeTemplate    *providerv1.ProviderSurfaceRuntime
}

func newConnectTarget(
	addMethod AddMethod,
	displayName, vendorID, cliID, surfaceID string,
	runtime *providerv1.ProviderSurfaceRuntime,
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
		vendorID,
		cliID,
		surfaceID,
		targetCredentialID,
		targetProviderID,
		runtime,
	), nil
}

func newConnectTargetWithIDs(
	addMethod AddMethod,
	displayName, vendorID, cliID, surfaceID, targetCredentialID, targetProviderID string,
	runtime *providerv1.ProviderSurfaceRuntime,
) *connectTarget {
	return &connectTarget{
		AddMethod:          addMethod,
		DisplayName:        strings.TrimSpace(displayName),
		VendorID:           strings.TrimSpace(vendorID),
		CLIID:              strings.TrimSpace(cliID),
		SurfaceID:          strings.TrimSpace(surfaceID),
		TargetCredentialID: strings.TrimSpace(targetCredentialID),
		TargetProviderID:   strings.TrimSpace(targetProviderID),
		RuntimeTemplate:    cloneProviderSurfaceRuntime(runtime),
	}
}

func (t *connectTarget) WithSharedIdentity(targetCredentialID, targetProviderID string) *connectTarget {
	if t == nil {
		return &connectTarget{}
	}
	return newConnectTargetWithIDs(
		t.AddMethod,
		t.DisplayName,
		t.VendorID,
		t.CLIID,
		t.SurfaceID,
		targetCredentialID,
		targetProviderID,
		t.RuntimeTemplate,
	)
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

func (t *connectTarget) callableRuntime() *providerv1.ProviderSurfaceRuntime {
	if t == nil {
		return nil
	}
	return cloneProviderSurfaceRuntime(t.RuntimeTemplate)
}

func (t *connectTarget) Provider(credentialID string) *providerv1.Provider {
	provider := &providerv1.Provider{
		ProviderId:  strings.TrimSpace(t.TargetProviderID),
		DisplayName: strings.TrimSpace(t.DisplayName),
		SurfaceId:   strings.TrimSpace(t.SurfaceID),
		Runtime:     t.callableRuntime(),
	}
	if cid := strings.TrimSpace(credentialID); cid != "" {
		provider.ProviderCredentialRef = &providerv1.ProviderCredentialRef{ProviderCredentialId: cid}
	}
	return provider
}


