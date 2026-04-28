package providers

import (
	"context"

	providerv1 "code-code.internal/go-contract/provider/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type providerMutationRuntime struct {
	repository  Store
	credentials credentialService
}

func newProviderMutationRuntime(repository Store, credentials credentialService) providerMutationRuntime {
	return providerMutationRuntime{repository: repository, credentials: credentials}
}

func (s *Service) mutationRuntime() providerMutationRuntime {
	return newProviderMutationRuntime(s.repository, s.credentials)
}

func (r providerMutationRuntime) Rename(ctx context.Context, projection *ProviderProjection, command UpdateProviderCommand) error {
	displayName, renameCredential, err := projection.Rename(command.DisplayName)
	if err != nil {
		return err
	}
	if _, err := r.repository.Update(ctx, projection.ID(), func(provider *providerv1.Provider) error {
		provider.DisplayName = displayName
		return nil
	}); err != nil {
		return err
	}
	if renameCredential {
		if err := r.credentials.Rename(ctx, projection.CredentialID(), displayName); err != nil {
			return err
		}
	}
	return r.renameObservabilityCredentialIfPresent(ctx, projection, displayName)
}

func (r providerMutationRuntime) Delete(ctx context.Context, projection *ProviderProjection) error {
	if err := projection.ValidateMutable(); err != nil {
		return err
	}
	if err := r.repository.Delete(ctx, projection.ID()); err != nil {
		return err
	}
	if projection.CredentialID() != "" {
		if err := r.credentials.Delete(ctx, projection.CredentialID()); err != nil {
			return err
		}
	}
	return r.deleteObservabilityCredentialIfPresent(ctx, projection)
}

func (r providerMutationRuntime) UpdateAPIKeyAuthentication(ctx context.Context, projection *ProviderProjection, command UpdateAPIKeyAuthenticationCommand) error {
	credential, err := projection.APIKeyAuthenticationCredential(command.APIKey)
	if err != nil {
		return err
	}
	_, err = r.credentials.UpdateAPIKey(ctx, *credential)
	return err
}

func (r providerMutationRuntime) UpdateObservabilityAuthentication(ctx context.Context, projection *ProviderProjection, command UpdateObservabilityAuthenticationCommand) error {
	credential, err := projection.ObservabilityCredential(command)
	if err != nil {
		return err
	}
	if credential == nil {
		return r.deleteObservabilityCredentialIfPresent(ctx, projection)
	}
	_, err = r.credentials.UpdateSession(ctx, *credential)
	return err
}

func (r providerMutationRuntime) renameObservabilityCredentialIfPresent(ctx context.Context, projection *ProviderProjection, displayName string) error {
	exists, err := r.observabilityCredentialExists(ctx, projection)
	if err != nil {
		return err
	}
	if projection.AuthKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API || !exists {
		return nil
	}
	return r.credentials.Rename(ctx, projection.ObservabilityCredentialID(), projection.observabilityCredentialDisplayName(displayName))
}

func (r providerMutationRuntime) deleteObservabilityCredentialIfPresent(ctx context.Context, projection *ProviderProjection) error {
	exists, err := r.observabilityCredentialExists(ctx, projection)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	err = r.credentials.Delete(ctx, projection.ObservabilityCredentialID())
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (r providerMutationRuntime) observabilityCredentialExists(ctx context.Context, projection *ProviderProjection) (bool, error) {
	if r.credentials == nil || projection == nil || projection.AuthKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
		return false, nil
	}
	return r.credentials.Exists(ctx, projection.ObservabilityCredentialID())
}
