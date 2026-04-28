package providerconnect

import (
	"strings"

	"code-code.internal/platform-k8s/internal/platform/resourcemeta"
)

type connectPlan struct {
	DisplayName        string
	VendorID           string
	TargetCredentialID string
	TargetProviderID   string
	Targets            []*connectTarget
}

func newConnectPlan(displayName, vendorID string, targets []*connectTarget) (*connectPlan, error) {
	targetCredentialID, err := resourcemeta.EnsureResourceID("", displayName, vendorID)
	if err != nil {
		return nil, err
	}
	targetProviderID, err := resourcemeta.EnsureResourceID("", displayName, vendorID+"-provider")
	if err != nil {
		return nil, err
	}
	items := make([]*connectTarget, 0, len(targets))
	for _, target := range targets {
		if target == nil {
			continue
		}
		items = append(items, target.WithSharedIdentity(targetCredentialID, targetProviderID))
	}
	return &connectPlan{
		DisplayName:        strings.TrimSpace(displayName),
		VendorID:           strings.TrimSpace(vendorID),
		TargetCredentialID: targetCredentialID,
		TargetProviderID:   targetProviderID,
		Targets:            items,
	}, nil
}

func (p *connectPlan) APIKeyCredential(apiKey string) *CredentialAPIKeyCreate {
	if p == nil {
		return nil
	}
	return &CredentialAPIKeyCreate{
		CredentialID: strings.TrimSpace(p.TargetCredentialID),
		DisplayName:  strings.TrimSpace(p.DisplayName),
		VendorID:     strings.TrimSpace(p.VendorID),
		APIKey:       strings.TrimSpace(apiKey),
	}
}
