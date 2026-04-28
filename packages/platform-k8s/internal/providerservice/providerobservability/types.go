package providerobservability

import (
	"context"
	"time"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

// Trigger identifies why an observability probe is requested.
type Trigger string

const (
	TriggerSchedule Trigger = "schedule"
	TriggerManual   Trigger = "manual"
	TriggerConnect  Trigger = "connect"
)

// OwnerKind identifies the host package family that owns one observability surface.
type OwnerKind string

const (
	OwnerKindCLI    OwnerKind = "cli"
	OwnerKindVendor OwnerKind = "vendor"
)

// ProbeOutcome is the normalized observability probe outcome.
type ProbeOutcome string

const (
	ProbeOutcomeExecuted    ProbeOutcome = "executed"
	ProbeOutcomeThrottled   ProbeOutcome = "throttled"
	ProbeOutcomeAuthBlocked ProbeOutcome = "auth_blocked"
	ProbeOutcomeUnsupported ProbeOutcome = "unsupported"
	ProbeOutcomeFailed      ProbeOutcome = "failed"
)

// ProbeTarget carries the provider account and owner package selected for one probe.
type ProbeTarget struct {
	ProviderID               string
	ProviderSurfaceBindingID string
	OwnerKind                OwnerKind
	OwnerID                  string
}

// ProbeResult is the normalized active observability probe result.
type ProbeResult struct {
	OwnerKind                OwnerKind
	OwnerID                  string
	ProviderID               string
	ProviderSurfaceBindingID string
	Outcome                  ProbeOutcome
	Message                  string
	Reason                   string
	LastAttemptAt            *time.Time
	NextAllowedAt            *time.Time
}

// Capability owns one active observability family such as CLI OAuth or vendor API key.
type Capability interface {
	OwnerKind() OwnerKind
	Supports(surface *managementv1.ProviderSurfaceBindingView) (ownerID string, ok bool)
	ProbeProvider(ctx context.Context, target ProbeTarget, trigger Trigger) (*ProbeResult, error)
}

type providerSurfaceBindingLister interface {
	ListProviderSurfaceBindings(ctx context.Context) ([]*managementv1.ProviderSurfaceBindingView, error)
}

// Service dispatches provider observability probes without owning credential material.
type Service struct {
	providerSurfaceBindings providerSurfaceBindingLister
	capabilities            []Capability
}

// Config groups provider observability dependencies.
type Config struct {
	ProviderSurfaceBindings providerSurfaceBindingLister
	Capabilities            []Capability
}
