package providerconnect

import (
	"fmt"
	"strings"

	providerv1 "code-code.internal/go-contract/provider/v1"
)

type sessionTargetSnapshot struct {
	AddMethod          AddMethod `json:"addMethod"`
	DisplayName        string    `json:"displayName"`
	CLIID              string    `json:"cliId"`
	ProviderSurfaceID  string    `json:"providerSurfaceId"`
	TargetCredentialID string    `json:"targetCredentialId"`
	TargetProviderID   string    `json:"targetProviderId"`
}

func newSessionTargetSnapshot(target *connectTarget) (sessionTargetSnapshot, error) {
	if target == nil {
		return sessionTargetSnapshot{}, fmt.Errorf("platformk8s/providerconnect: session target is nil")
	}
	return sessionTargetSnapshot{
		AddMethod:          target.AddMethod,
		DisplayName:        strings.TrimSpace(target.DisplayName),
		CLIID:              strings.TrimSpace(target.CLIID),
		ProviderSurfaceID:  strings.TrimSpace(target.SurfaceID),
		TargetCredentialID: strings.TrimSpace(target.TargetCredentialID),
		TargetProviderID:   strings.TrimSpace(target.TargetProviderID),
	}, nil
}

func (s sessionTargetSnapshot) needsFinalize(connectedSurfaceID string) bool {
	return strings.TrimSpace(s.ProviderSurfaceID) != "" && strings.TrimSpace(connectedSurfaceID) == ""
}

func (s sessionTargetSnapshot) models() ([]*providerv1.ProviderModel, error) {
	return nil, nil
}

func (s sessionTargetSnapshot) target(models []*providerv1.ProviderModel) *connectTarget {
	return newConnectTargetWithIDs(
		s.AddMethod,
		s.DisplayName,
		s.CLIID,
		s.ProviderSurfaceID,
		s.TargetCredentialID,
		s.TargetProviderID,
		models,
	)
}
