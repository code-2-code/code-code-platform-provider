package providers

import (
	"context"

	providerv1 "code-code.internal/go-contract/provider/v1"
)

type providerMutationRuntime struct {
	repository Store
}

func newProviderMutationRuntime(repository Store) providerMutationRuntime {
	return providerMutationRuntime{repository: repository}
}

func (s *Service) mutationRuntime() providerMutationRuntime {
	return newProviderMutationRuntime(s.repository)
}

func (r providerMutationRuntime) Rename(ctx context.Context, projection *ProviderProjection, command UpdateProviderCommand) error {
	displayName, err := projection.Rename(command.DisplayName)
	if err != nil {
		return err
	}
	_, err = r.repository.Update(ctx, projection.ID(), func(provider *providerv1.Provider) error {
		provider.DisplayName = displayName
		return nil
	})
	return err
}

func (r providerMutationRuntime) Delete(ctx context.Context, projection *ProviderProjection) error {
	if err := projection.ValidateMutable(); err != nil {
		return err
	}
	return r.repository.Delete(ctx, projection.ID())
}
