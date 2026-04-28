package providerobservability

import (
	"testing"
)

func TestParseInternlmStatisticsGaugeRows(t *testing.T) {
	body := []byte(`{
		"code": 0,
		"msg": "",
		"data": {
			"total": {"calls": 100, "input_tokens": 50000, "output_tokens": 20000},
			"last_seven_day": {"calls": 10, "input_tokens": 5000, "output_tokens": 2000},
			"seven_day_diff": {"calls": 0, "input_tokens": 0, "output_tokens": 0},
			"month_used": {"calls": 42, "input_tokens": 120000, "output_tokens": 45000},
			"month_quota": {"calls": 0, "input_tokens": 9000000, "output_tokens": 3000000}
		}
	}`)

	rows, err := parseInternlmStatisticsGaugeRows(body)
	if err != nil {
		t.Fatalf("parseInternlmStatisticsGaugeRows() error = %v", err)
	}
	// 2 token types (input, output), each 4 source rows = 8
	if len(rows) != 8 {
		t.Errorf("parseInternlmStatisticsGaugeRows() returned %d rows, want 8", len(rows))
	}

	// Verify input tokens limit = 9000000
	for _, row := range rows {
		if row.MetricName == internlmDailyQuotaLimitMetric &&
			row.Labels["resource"] == "tokens" &&
			row.Labels["token_type"] == "input" {
			if row.Value != 9000000 {
				t.Errorf("input_tokens limit = %v, want 9000000", row.Value)
			}
			break
		}
	}

	// Verify output tokens usage = 45000
	for _, row := range rows {
		if row.MetricName == internlmDailyQuotaUsageMetric &&
			row.Labels["resource"] == "tokens" &&
			row.Labels["token_type"] == "output" {
			if row.Value != 45000 {
				t.Errorf("output_tokens usage = %v, want 45000", row.Value)
			}
			break
		}
	}

	// Verify output tokens remaining = 2955000
	for _, row := range rows {
		if row.MetricName == internlmDailyQuotaRemainingMetric &&
			row.Labels["resource"] == "tokens" &&
			row.Labels["token_type"] == "output" {
			if row.Value != 2955000 {
				t.Errorf("output_tokens remaining = %v, want 2955000", row.Value)
			}
			break
		}
	}
}

func TestParseInternlmStatisticsGaugeRows_ZeroQuota(t *testing.T) {
	body := []byte(`{
		"code": 0,
		"data": {
			"month_used": {"calls": 0, "input_tokens": 0, "output_tokens": 0},
			"month_quota": {"calls": 0, "input_tokens": 0, "output_tokens": 0}
		}
	}`)
	rows, err := parseInternlmStatisticsGaugeRows(body)
	if err != nil {
		t.Fatalf("parseInternlmStatisticsGaugeRows() error = %v", err)
	}
	// Zero quotas should produce no rows
	if len(rows) != 0 {
		t.Errorf("parseInternlmStatisticsGaugeRows() returned %d rows, want 0", len(rows))
	}
}

func TestParseInternlmStatisticsGaugeRows_ErrorCode(t *testing.T) {
	body := []byte(`{"code": 401, "msg": "unauthorized"}`)
	_, err := parseInternlmStatisticsGaugeRows(body)
	if err == nil {
		t.Fatal("parseInternlmStatisticsGaugeRows() error = nil, want error")
	}
}

func TestParseInternlmStatisticsGaugeRows_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	_, err := parseInternlmStatisticsGaugeRows(body)
	if err == nil {
		t.Fatal("parseInternlmStatisticsGaugeRows() error = nil, want error")
	}
}
