package providers

import (
	"fmt"

	anthropicprovider "code-code.internal/platform-k8s/internal/providerservice/providers/anthropic"
	geminiprovider "code-code.internal/platform-k8s/internal/providerservice/providers/gemini"
	"code-code.internal/platform-k8s/internal/providerservice/providers/openaicompatible"
)

// RegisterBuiltins registers implementation-owned provider runtimes that ship
// with the platform.
func RegisterBuiltins(lookup *Lookup) error {
	if lookup == nil {
		return fmt.Errorf("platformk8s/providers: lookup is nil")
	}
	if err := lookup.Register(openaicompatible.NewProvider()); err != nil {
		return err
	}
	if err := lookup.Register(geminiprovider.NewProvider()); err != nil {
		return err
	}
	return lookup.Register(anthropicprovider.NewProvider())
}
