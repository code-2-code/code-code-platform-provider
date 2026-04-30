package providerconnect

// ProviderPhase is the providerconnect-owned provider phase.
type ProviderPhase string

const (
	ProviderPhaseUnspecified   ProviderPhase = ""
	ProviderPhaseReady         ProviderPhase = "ready"
	ProviderPhaseInvalidConfig ProviderPhase = "invalid_config"
	ProviderPhaseRefreshing    ProviderPhase = "refreshing"
	ProviderPhaseStale         ProviderPhase = "stale"
	ProviderPhaseError         ProviderPhase = "error"
)
