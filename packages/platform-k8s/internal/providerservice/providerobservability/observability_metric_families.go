package providerobservability

const (
	// Unified canonical metric families for provider observability.
	// Names stay vendor-neutral and owner-neutral; vendor/cli identity lives in labels.
	providerQuotaLimitMetric                    = "gen_ai.provider.quota.limit"
	providerQuotaUsageMetric                    = "gen_ai.provider.quota.usage"
	providerQuotaRemainingMetric                = "gen_ai.provider.quota.remaining"
	providerQuotaRemainingFractionPercentMetric = "gen_ai.provider.quota.remaining.fraction.percent"
	providerQuotaUsageFractionPercentMetric     = "gen_ai.provider.quota.usage.fraction.percent"
	providerQuotaResetTimestampMetric           = "gen_ai.provider.quota.reset.timestamp.seconds"
	providerRuntimeQuotaLimitMetric             = "gen_ai.provider.runtime.quota.limit"
	providerRuntimeQuotaRemainingMetric         = "gen_ai.provider.runtime.quota.remaining"
	providerRuntimeRateLimitLimitMetric         = "gen_ai.provider.runtime.rate_limit.limit"
	providerRuntimeRateLimitRemainingMetric     = "gen_ai.provider.runtime.rate_limit.remaining"
	providerUsageRequestsMetric                 = "gen_ai.provider.usage.requests.count"
	providerUsageTokensMetric                   = "gen_ai.provider.usage.tokens.count"
	providerUsageCostUSDMetric                  = "gen_ai.provider.usage.cost.usd"
)

const (
	ownerKindLabel  = "owner_kind"
	ownerIDLabel    = "owner_id"
	ownerKindCLI    = "cli"
	ownerKindVendor = "vendor"
)
