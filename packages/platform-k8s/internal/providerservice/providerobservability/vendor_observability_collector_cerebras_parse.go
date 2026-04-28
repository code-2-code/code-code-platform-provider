package providerobservability

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// cerebrasUsageQuota represents one model's effective quota limits from
// ListOrganizationEffectiveQuotas.
type cerebrasUsageQuota struct {
	ModelID              string `json:"modelId"`
	RegionID             string `json:"regionId"`
	RequestsPerMinute    string `json:"requestsPerMinute"`
	TokensPerMinute      string `json:"tokensPerMinute"`
	RequestsPerHour      string `json:"requestsPerHour"`
	TokensPerHour        string `json:"tokensPerHour"`
	RequestsPerDay       string `json:"requestsPerDay"`
	TokensPerDay         string `json:"tokensPerDay"`
	TotalTokensPerMinute string `json:"totalTokensPerMinute"`
	TotalTokensPerHour   string `json:"totalTokensPerHour"`
	TotalTokensPerDay    string `json:"totalTokensPerDay"`
}

// cerebrasOrganizationUsage represents one model's current usage from
// ListOrganizationUsage.
type cerebrasOrganizationUsage struct {
	ModelID  string `json:"modelId"`
	RegionID string `json:"regionId"`
	RPM      string `json:"rpm"`
	TPM      string `json:"tpm"`
	RPH      string `json:"rph"`
	TPH      string `json:"tph"`
	RPD      string `json:"rpd"`
	TPD      string `json:"tpd"`
}

// cerebrasOrganization represents one organization from ListMyOrganizations.
type cerebrasOrganization struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// parseCerebrasOrganizations extracts all organizations from the
// ListMyOrganizations GraphQL response, preferring active organizations first.
func parseCerebrasOrganizations(body []byte) ([]cerebrasOrganization, error) {
	var resp struct {
		Data struct {
			ListMyOrganizations []cerebrasOrganization `json:"ListMyOrganizations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: decode organizations: %w", err)
	}
	active := make([]cerebrasOrganization, 0, len(resp.Data.ListMyOrganizations))
	inactive := make([]cerebrasOrganization, 0, len(resp.Data.ListMyOrganizations))
	for _, org := range resp.Data.ListMyOrganizations {
		org.ID = strings.TrimSpace(org.ID)
		org.Name = strings.TrimSpace(org.Name)
		org.State = strings.TrimSpace(org.State)
		if org.ID == "" {
			continue
		}
		if strings.EqualFold(org.State, "active") {
			active = append(active, org)
			continue
		}
		inactive = append(inactive, org)
	}
	organizations := append(active, inactive...)
	if len(organizations) == 0 {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: no organizations found")
	}
	return organizations, nil
}

// parseCerebrasGraphQLGaugeRows converts quota limits and usage data into
// metric gauge rows.
func parseCerebrasGraphQLGaugeRows(
	quotaBody []byte,
	usageBody []byte,
	orgID string,
	orgName string,
) ([]VendorObservabilityMetricRow, error) {
	var quotaResp struct {
		Data struct {
			ListOrganizationEffectiveQuotas []cerebrasUsageQuota `json:"ListOrganizationEffectiveQuotas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(quotaBody, &quotaResp); err != nil {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: decode quotas: %w", err)
	}
	if len(quotaResp.Data.ListOrganizationEffectiveQuotas) == 0 {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: no quota entries returned")
	}

	// Build usage lookup: modelId+regionId → usage
	usageLookup := map[string]*cerebrasOrganizationUsage{}
	if len(usageBody) > 0 {
		var usageResp struct {
			Data struct {
				ListOrganizationUsage []cerebrasOrganizationUsage `json:"ListOrganizationUsage"`
			} `json:"data"`
		}
		if err := json.Unmarshal(usageBody, &usageResp); err == nil {
			for i := range usageResp.Data.ListOrganizationUsage {
				u := &usageResp.Data.ListOrganizationUsage[i]
				key := strings.TrimSpace(u.ModelID) + "|" + strings.TrimSpace(u.RegionID)
				usageLookup[key] = u
			}
		}
	}

	var rows []VendorObservabilityMetricRow
	for _, q := range quotaResp.Data.ListOrganizationEffectiveQuotas {
		modelID := strings.TrimSpace(q.ModelID)
		regionID := strings.TrimSpace(q.RegionID)
		usageKey := modelID + "|" + regionID
		usage := usageLookup[usageKey]

		type windowEntry struct {
			window   string
			resource string
			limitStr string
			usageStr string
		}
		entries := []windowEntry{
			{"minute", "requests", q.RequestsPerMinute, cerebrasUsageField(usage, "rpm")},
			{"minute", "tokens", q.TokensPerMinute, cerebrasUsageField(usage, "tpm")},
			{"hour", "requests", q.RequestsPerHour, cerebrasUsageField(usage, "rph")},
			{"hour", "tokens", q.TokensPerHour, cerebrasUsageField(usage, "tph")},
			{"day", "requests", q.RequestsPerDay, cerebrasUsageField(usage, "rpd")},
			{"day", "tokens", q.TokensPerDay, cerebrasUsageField(usage, "tpd")},
		}

		for _, entry := range entries {
			limit := cerebrasParseInt64(entry.limitStr)
			if limit <= 0 {
				continue
			}
			used := cerebrasParseInt64(entry.usageStr)
			remaining := limit - used
			if remaining < 0 {
				remaining = 0
			}

			labels := map[string]string{
				"model_id": modelID,
				"window":   entry.window,
				"resource": entry.resource,
				"org_id":   strings.TrimSpace(orgID),
			}
			if trimmedOrgName := strings.TrimSpace(orgName); trimmedOrgName != "" {
				labels["org_name"] = trimmedOrgName
			}
			if regionID != "" {
				labels["region_id"] = regionID
			}

			rows = append(rows,
				VendorObservabilityMetricRow{MetricName: cerebrasQuotaLimitMetric, Labels: labels, Value: float64(limit)},
				VendorObservabilityMetricRow{MetricName: cerebrasQuotaUsageMetric, Labels: labels, Value: float64(used)},
				VendorObservabilityMetricRow{MetricName: cerebrasQuotaRemainingMetric, Labels: labels, Value: float64(remaining)},
			)
			if limit > 0 {
				rows = append(rows, VendorObservabilityMetricRow{
					MetricName: cerebrasQuotaUsagePercentMetric,
					Labels:     labels,
					Value:      float64(used) / float64(limit) * 100.0,
				})
			}
		}
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: cerebras quotas: no metric rows produced from graphql data")
	}
	return rows, nil
}

// cerebrasUsageField extracts the named field from an OrganizationUsage entry.
func cerebrasUsageField(usage *cerebrasOrganizationUsage, field string) string {
	if usage == nil {
		return ""
	}
	switch field {
	case "rpm":
		return usage.RPM
	case "tpm":
		return usage.TPM
	case "rph":
		return usage.RPH
	case "tph":
		return usage.TPH
	case "rpd":
		return usage.RPD
	case "tpd":
		return usage.TPD
	default:
		return ""
	}
}

// cerebrasParseInt64 parses a string to int64, returning 0 for empty, negative,
// or unparseable values.
func cerebrasParseInt64(s string) int64 {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0
	}
	v, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
