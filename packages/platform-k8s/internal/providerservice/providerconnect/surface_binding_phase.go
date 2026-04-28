package providerconnect

// ProviderSurfaceBindingPhase is the providerconnect-owned provider surface phase.
type ProviderSurfaceBindingPhase string

const (
	ProviderSurfaceBindingPhaseUnspecified   ProviderSurfaceBindingPhase = ""
	ProviderSurfaceBindingPhaseReady         ProviderSurfaceBindingPhase = "ready"
	ProviderSurfaceBindingPhaseInvalidConfig ProviderSurfaceBindingPhase = "invalid_config"
	ProviderSurfaceBindingPhaseRefreshing    ProviderSurfaceBindingPhase = "refreshing"
	ProviderSurfaceBindingPhaseStale         ProviderSurfaceBindingPhase = "stale"
	ProviderSurfaceBindingPhaseError         ProviderSurfaceBindingPhase = "error"
)
