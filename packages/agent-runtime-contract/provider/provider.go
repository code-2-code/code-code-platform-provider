// Package provider defines the runtime behavior contract used by the platform
// to drive provider implementations.
package provider

import (
	"context"

	credentialcontract "code-code.internal/agent-runtime-contract/credential"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
)

// ProviderSurface describes one stable provider capability surface.
type ProviderSurface = supportv1.Surface

// ResolvedProviderModel describes the final provider-routed model selected for
// one call.
type ResolvedProviderModel = providerv1.ResolvedProviderModel

// Provider is the entry point implemented by one provider surface.
type Provider interface {
	// Surface returns the stable provider surface metadata.
	Surface() *ProviderSurface

	// NewRuntime creates one runtime bound to the supplied configured provider and
	// resolved credential. credential may be nil when the bound surface does
	// not reference a credential.
	NewRuntime(provider *providerv1.Provider, credential *credentialcontract.ResolvedCredential) (ProviderRuntime, error)
}

// ProviderRuntime is the platform-driven runtime for one provider surface.
type ProviderRuntime interface {
	// HealthCheck reports whether the runtime is still healthy enough to serve requests.
	HealthCheck(ctx context.Context) error

	// ListModels returns the current provider-callable models for the bound surface.
	ListModels(ctx context.Context) ([]*providerv1.ProviderModel, error)

	// Close releases runtime-owned resources.
	Close(ctx context.Context) error
}

// ProviderRegistry lists the configured providers available to the platform.
type ProviderRegistry interface {
	// ListProviders returns the currently selectable configured providers.
	ListProviders(ctx context.Context) ([]*providerv1.Provider, error)
}
