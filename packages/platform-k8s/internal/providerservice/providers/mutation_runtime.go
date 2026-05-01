package providers

import (
	"context"
	"fmt"

	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
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

func (r providerMutationRuntime) ApplyModelCatalog(ctx context.Context, projection *ProviderProjection, models []*providerv1.ProviderModel) error {
	if err := providerv1.ValidateProviderModels(models); err != nil {
		return err
	}
	_, err := r.repository.Update(ctx, projection.ID(), func(provider *providerv1.Provider) error {
		provider.Models = cloneProviderModels(models)
		return nil
	})
	return err
}

func (r providerMutationRuntime) ApplyProbeStatus(ctx context.Context, projection *ProviderProjection, kind providerv1.ProviderProbeKind, state *providerv1.ProviderProbeRunState) error {
	if state == nil {
		return fmt.Errorf("platformk8s/providers: provider probe state is nil")
	}
	_, err := r.repository.Update(ctx, projection.ID(), func(provider *providerv1.Provider) error {
		status := provider.GetProbeStatus()
		if status == nil {
			status = &providerv1.ProviderProbeStatus{}
			provider.ProbeStatus = status
		}
		next := proto.Clone(state).(*providerv1.ProviderProbeRunState)
		switch kind {
		case providerv1.ProviderProbeKind_PROVIDER_PROBE_KIND_MODEL_CATALOG:
			status.ModelCatalog = next
		case providerv1.ProviderProbeKind_PROVIDER_PROBE_KIND_QUOTA:
			status.Quota = next
		default:
			return fmt.Errorf("platformk8s/providers: provider probe kind is unspecified")
		}
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
