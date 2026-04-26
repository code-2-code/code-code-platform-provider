// Package provider defines the runtime behavior contract used by the platform
// to drive provider implementations.
package provider

import (
	"context"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

// ProviderSurface describes one stable provider capability surface.
type ProviderSurface = providerv1.ProviderSurface

// ProviderSurfaceBinding describes one configured callable surface on a provider account.
type ProviderSurfaceBinding = providerv1.ProviderSurfaceBinding

// ProviderModelCatalog describes the models currently available through one surface.
type ProviderModelCatalog = providerv1.ProviderModelCatalog

// ResolvedProviderModel describes the final provider-routed model selected for
// one call.
type ResolvedProviderModel = providerv1.ResolvedProviderModel

// Provider is the entry point implemented by one provider surface.
type Provider interface {
	// Surface returns the stable provider surface metadata.
	Surface() *ProviderSurface

	// NewRuntime creates one runtime bound to the supplied account surface and
	// resolved credential. credential may be nil when the bound surface does
	// not reference a credential.
	NewRuntime(surface *ProviderSurfaceBinding, credential *credentialcontract.ResolvedCredential) (ProviderRuntime, error)
}

// ProviderRuntime is the platform-driven runtime for one provider surface.
type ProviderRuntime interface {
	// HealthCheck reports whether the runtime is still healthy enough to serve requests.
	HealthCheck(ctx context.Context) error

	// ListModels returns the current model catalog for the bound surface.
	ListModels(ctx context.Context) (*ProviderModelCatalog, error)

	// Close releases runtime-owned resources.
	Close(ctx context.Context) error
}

// ProviderRegistry lists the provider account surfaces available to the platform.
type ProviderRegistry interface {
	// ListProviderSurfaceBindings returns the currently selectable provider account surfaces.
	ListProviderSurfaceBindings(ctx context.Context) ([]*ProviderSurfaceBinding, error)
}
