package providerobservability

import (
	"context"
	"time"

	providerv1 "code-code.internal/go-contract/provider/v1"
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
	OwnerKindSurface OwnerKind = "surface"
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

// ProbeTarget carries the provider account and surface schema selected for one probe.
type ProbeTarget struct {
	ProviderID string
	SurfaceID  string
	OwnerKind  OwnerKind
	SchemaID   string
}

// ProbeResult is the normalized active observability probe result.
type ProbeResult struct {
	OwnerKind     OwnerKind
	SchemaID      string
	ProviderID    string
	SurfaceID     string
	Outcome       ProbeOutcome
	Message       string
	Reason        string
	LastAttemptAt *time.Time
	NextAllowedAt *time.Time
}

// Capability owns one active observability family such as CLI OAuth or vendor API key.
type Capability interface {
	OwnerKind() OwnerKind
	Supports(ctx context.Context, provider *providerv1.Provider) (schemaID string, ok bool)
	ProbeProvider(ctx context.Context, target ProbeTarget, trigger Trigger) (*ProbeResult, error)
}

type providerLister interface {
	List(ctx context.Context) ([]*providerv1.Provider, error)
}

// Service dispatches provider observability probes without owning credential material.
type Service struct {
	providers    providerLister
	capabilities []Capability
}

// Config groups provider observability dependencies.
type Config struct {
	Providers    providerLister
	Capabilities []Capability
}
