package protocolruntime

import (
	"context"
	"sync"
	"time"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

// BaseRuntime provides shared runtime lifecycle logic for provider
// implementations that expose one configured provider model set.
type BaseRuntime struct {
	Provider   *providerv1.Provider
	Credential *credentialcontract.ResolvedCredential
	Now        func() time.Time

	mu     sync.RWMutex
	models []*providerv1.ProviderModel
}

// HealthCheck reports whether the runtime surface config is valid enough to
// attempt a call. Model catalog discovery is asynchronous and is not a health
// gate.
func (r *BaseRuntime) HealthCheck(ctx context.Context) error {
	if r == nil || r.Provider == nil {
		return nil
	}
	return providerv1.ValidateProvider(r.Provider)
}

// ListModels returns the configured models for the bound provider.
func (r *BaseRuntime) ListModels(ctx context.Context) ([]*providerv1.ProviderModel, error) {
	r.mu.RLock()
	if r.models != nil {
		cached := cloneProviderModels(r.models)
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()
	return r.providerModels(ctx)
}

// providerModels returns the pre-configured provider models.
func (r *BaseRuntime) providerModels(_ context.Context) ([]*providerv1.ProviderModel, error) {
	models := cloneProviderModels(r.Provider.GetModels())
	if len(models) > 0 {
		if err := providerv1.ValidateProviderModels(models); err != nil {
			return nil, err
		}
	}
	r.mu.Lock()
	r.models = cloneProviderModels(models)
	r.mu.Unlock()
	return models, nil
}

// Close releases runtime-owned resources.
func (r *BaseRuntime) Close(_ context.Context) error {
	return nil
}

func cloneProviderModels(models []*providerv1.ProviderModel) []*providerv1.ProviderModel {
	if models == nil {
		return nil
	}
	out := make([]*providerv1.ProviderModel, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		out = append(out, proto.Clone(model).(*providerv1.ProviderModel))
	}
	return out
}
