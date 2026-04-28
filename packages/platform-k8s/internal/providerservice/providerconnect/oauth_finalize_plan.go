package providerconnect

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type oauthFinalizePlan struct {
	target       *connectTarget
	credentialID string
}

func newOAuthFinalizePlan(record *sessionRecord, oauthState *credentialv1.OAuthAuthorizationSessionState) (*oauthFinalizePlan, error) {
	if record == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: session record is nil")
	}
	if oauthState == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: oauth session state is nil")
	}
	credentialID := strings.TrimSpace(oauthState.GetStatus().GetImportedCredential().GetCredentialId())
	if credentialID == "" {
		credentialID = strings.TrimSpace(oauthState.GetSpec().GetTargetCredentialId())
	}
	if credentialID == "" {
		return nil, domainerror.NewValidation(
			"platformk8s/providerconnect: oauth session %q completed without imported credential",
			oauthState.GetSpec().GetSessionId(),
		)
	}
	runtime, err := record.runtime()
	if err != nil {
		return nil, err
	}
	return &oauthFinalizePlan{
		target:       record.target(runtime),
		credentialID: credentialID,
	}, nil
}

func (p *oauthFinalizePlan) CredentialID() string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(p.credentialID)
}

func (p *oauthFinalizePlan) TargetSurfaceID() string {
	if p == nil || p.target == nil {
		return ""
	}
	return p.target.SurfaceID
}

func (p *oauthFinalizePlan) CreateProviderSurfaceBinding() *providerv1.ProviderSurfaceBinding {
	if p == nil || p.target == nil {
		return &providerv1.ProviderSurfaceBinding{}
	}
	return p.target.ProviderSurfaceBinding(p.CredentialID())
}

func (p *oauthFinalizePlan) CreateProvider() *providerv1.Provider {
	if p == nil || p.target == nil {
		return &providerv1.Provider{}
	}
	return p.target.Provider(p.CredentialID())
}

func (p *oauthFinalizePlan) ValidateExisting(existing *ProviderSurfaceBindingView) error {
	surface := p.CreateProviderSurfaceBinding()
	if existing == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: existing provider surface binding is nil")
	}
	if surface == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface binding is nil")
	}
	if err := validateSurfaceFieldMatch(surface.GetSurfaceId(), "surface_id", existing.GetSurfaceId(), surface.GetSurfaceId()); err != nil {
		return err
	}
	if err := validateSurfaceFieldMatch(surface.GetSurfaceId(), "provider_id", existing.GetProviderId(), p.target.TargetProviderID); err != nil {
		return err
	}
	if err := validateSurfaceFieldMatch(
		surface.GetSurfaceId(),
		"provider_credential_id",
		existing.GetProviderCredentialId(),
		surface.GetProviderCredentialRef().GetProviderCredentialId(),
	); err != nil {
		return err
	}
	if !proto.Equal(existing.GetRuntime(), surface.GetRuntime()) {
		return domainerror.NewAlreadyExists(
			"platformk8s/providerconnect: provider surface binding %q already exists with different runtime",
			surface.GetSurfaceId(),
		)
	}
	return nil
}

func validateSurfaceFieldMatch(surfaceID, field, current, next string) error {
	if strings.TrimSpace(current) == strings.TrimSpace(next) {
		return nil
	}
	return domainerror.NewAlreadyExists(
		"platformk8s/providerconnect: provider surface binding %q already exists with different %s",
		surfaceID,
		field,
	)
}
