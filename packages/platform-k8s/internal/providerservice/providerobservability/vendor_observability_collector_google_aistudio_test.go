package providerobservability

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDecodeGoogleAIStudioRPCBodySupportsDirectAndBase64(t *testing.T) {
	direct, err := decodeGoogleAIStudioRPCBody([]byte(`[[["gemini-2.5-flash"]]]`))
	if err != nil {
		t.Fatalf("decodeGoogleAIStudioRPCBody(direct) error = %v", err)
	}
	rows, err := googleAIStudioPayloadRows(direct)
	if err != nil {
		t.Fatalf("googleAIStudioPayloadRows(direct) error = %v", err)
	}
	row, ok := googleAIStudioPayloadRow(rows[0])
	if !ok || googleAIStudioStringAt(row, 0) != "gemini-2.5-flash" {
		t.Fatalf("decoded direct row = %#v, want gemini-2.5-flash", rows[0])
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(`[[["projects/500942601540","gen-lang-client-123"]]]`))
	wrapped, err := decodeGoogleAIStudioRPCBody([]byte(`"` + encoded + `"`))
	if err != nil {
		t.Fatalf("decodeGoogleAIStudioRPCBody(base64) error = %v", err)
	}
	rows, err = googleAIStudioPayloadRows(wrapped)
	if err != nil {
		t.Fatalf("googleAIStudioPayloadRows(base64) error = %v", err)
	}
	row, ok = googleAIStudioPayloadRow(rows[0])
	if !ok || googleAIStudioStringAt(row, 1) != "gen-lang-client-123" {
		t.Fatalf("decoded base64 row = %#v, want gen-lang-client-123", rows[0])
	}
}

func TestResolveGoogleAIStudioProjectSupportsClientAndNumericIDs(t *testing.T) {
	payload := []any{
		[]any{
			[]any{"projects/500942601540", "gen-lang-client-123", "Default Gemini Project", []any{}, 1, 20},
		},
	}
	for _, input := range []string{"gen-lang-client-123", "projects/500942601540", "500942601540"} {
		project, err := resolveGoogleAIStudioProject(payload, input)
		if err != nil {
			t.Fatalf("resolveGoogleAIStudioProject(%q) error = %v", input, err)
		}
		if got, want := project.Path, "projects/500942601540"; got != want {
			t.Fatalf("project path = %q, want %q", got, want)
		}
		if got, want := project.Tier, "FREE"; got != want {
			t.Fatalf("project tier = %q, want %q", got, want)
		}
	}
}

func TestNormalizeGoogleAIStudioProjectPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{
			name:   "projects path",
			input:  "projects/946397203396",
			want:   "projects/946397203396",
			wantOK: true,
		},
		{
			name:   "numeric project id",
			input:  "946397203396",
			want:   "projects/946397203396",
			wantOK: true,
		},
		{
			name:   "trim spaces",
			input:  "  projects/946397203396  ",
			want:   "projects/946397203396",
			wantOK: true,
		},
		{
			name:   "client project id",
			input:  "gen-lang-client-0346413999",
			want:   "",
			wantOK: false,
		},
		{
			name:   "empty",
			input:  "",
			want:   "",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := normalizeGoogleAIStudioProjectPath(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("normalizeGoogleAIStudioProjectPath(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestParseGoogleAIStudioMetricTimeSeriesUsage(t *testing.T) {
	t.Parallel()

	payload, err := decodeGoogleAIStudioRPCBody([]byte(`[[[[[["1776663651",116580000],["15"]]],"gemini-2.5-flash"]],["1776660051",116580000],["1776663651",116580000],["60"]]`))
	if err != nil {
		t.Fatalf("decodeGoogleAIStudioRPCBody() error = %v", err)
	}
	usage := parseGoogleAIStudioMetricTimeSeriesUsage(payload, map[string]struct{}{"gemini-2.5-flash": {}})
	if got, want := usage["gemini-2.5-flash"], 15.0; got != want {
		t.Fatalf("usage[gemini-2.5-flash] = %v, want %v", got, want)
	}
}

func TestParseGoogleAIStudioRateLimitsSupportsGemmaModels(t *testing.T) {
	t.Parallel()

	models, err := parseGoogleAIStudioRateLimits(
		[]any{
			[]any{
				[]any{"gemma-3-1b", 20, 1, 2, []any{"14400"}, 1},
				[]any{"gemini-embedding-1.0", 20, 9, 2, []any{"1000"}, 1},
			},
		},
		googleAIStudioTierHint{TierCode: 20},
		map[string]googleAIStudioQuotaModelMeta{
			"gemma-3-1b":           {ModelID: "gemma-3-1b", CategoryCode: 1},
			"gemini-embedding-1.0": {ModelID: "gemini-embedding-1.0", CategoryCode: 1},
		},
	)
	if err != nil {
		t.Fatalf("parseGoogleAIStudioRateLimits() error = %v", err)
	}
	if got, want := len(models), 1; got != want {
		t.Fatalf("model count = %d, want %d", got, want)
	}
	if got, want := models[0].ModelID, "gemma-3-1b"; got != want {
		t.Fatalf("models[0].ModelID = %q, want %q", got, want)
	}
	if got, want := models[0].Category, googleAIStudioGemmaModelCategory; got != want {
		t.Fatalf("models[0].Category = %q, want %q", got, want)
	}
}

func TestGoogleAIStudioObservabilityCollectorCollect(t *testing.T) {
	requestBodies := map[string]string{}
	var metricTimeSeriesBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty before egress auth injection", got)
		}
		if got := r.Header.Get("Cookie"); got != "" {
			t.Fatalf("Cookie = %q, want empty before egress auth injection", got)
		}
		if got := r.Header.Get("X-Goog-Api-Key"); got != "" {
			t.Fatalf("X-Goog-Api-Key = %q, want empty before egress auth injection", got)
		}
		if got, want := r.Header.Get("Origin"), googleAIStudioDefaultOrigin; got != want {
			t.Fatalf("Origin = %q, want %q", got, want)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll(body) error = %v", err)
		}
		requestBodies[r.URL.Path] = string(body)
		switch r.URL.Path {
		case "/ListCloudProjects":
			_ = json.NewEncoder(w).Encode([]any{
				[]any{
					[]any{"projects/500942601540", "gen-lang-client-123", "Default Gemini Project", []any{}, 1, 20},
				},
			})
		case "/ListQuotaModels":
			_ = json.NewEncoder(w).Encode([]any{
				[]any{
					[]any{"gemini-2.5-flash", nil, 4, "gemini-2.5", []any{[]any{"models/gemini-2.5-flash", []any{1, 2, 3}, []any{8, 10}}}, "gemini-2.5-flash"},
					[]any{"gemini-2.5-pro", nil, 4, "gemini-2.5", []any{[]any{"models/gemini-2.5-pro", []any{1, 2, 3}, []any{8, 10}}}, "gemini-2.5-pro"},
					[]any{"veo-3-generate", nil, 3, "default", []any{[]any{"models/veo-3.1-generate-preview", nil, []any{15}}}, "veo-3.1"},
				},
			})
		case "/ListModelRateLimits":
			_ = json.NewEncoder(w).Encode([]any{
				[]any{
					[]any{"gemini-2.5-flash", 20, 2, 1, []any{"250000"}, 4},
					[]any{"gemini-2.5-flash", 20, 2, 2, []any{"9000000"}, 4},
					[]any{"gemini-2.5-flash", 20, 1, 1, []any{"5"}, 4},
					[]any{"gemini-2.5-flash", 20, 1, 2, []any{"20"}, 4},
					[]any{"gemini-2.5-flash", 30, 1, 1, []any{"1000"}, 4},
					[]any{"gemini-2.5-flash-tts", 20, 2, 1, []any{"10000"}, 3},
					[]any{"gemini-2.5-flash-tts", 20, 1, 2, []any{"10"}, 3},
					[]any{"gemini-2.5-pro", 20, 2, 1, []any{"0"}, 4},
					[]any{"gemini-2.5-pro", 20, 1, 1, []any{"0"}, 4},
					[]any{"gemini-2.5-pro", 20, 13, 2, []any{"500"}, 7, 1},
					[]any{"veo-3-generate", 20, 6, 1, []any{"0"}, 3, 1},
					[]any{"veo-3-generate", 20, 6, 2, []any{"0"}, 3, 1},
					[]any{"gemini-2.5", 20, 12, 2, []any{"1500"}, 6},
				},
			})
		case "/FetchMetricTimeSeries":
			metricTimeSeriesBodies = append(metricTimeSeriesBodies, string(body))
			switch string(body) {
			case `[null,null,null,null,3,null,2,"projects/500942601540",null,[20],[1]]`:
				_ = json.NewEncoder(w).Encode([]any{
					[]any{
						[]any{
							[]any{
								[]any{
									[]any{"1776663651", 203285000},
									[]any{"3"},
								},
							},
							"gemini-2.5-flash",
						},
					},
					[]any{"1776660051", 203285000},
					[]any{"1776663651", 203285000},
					[]any{"60"},
				})
			case `[null,null,null,null,3,null,2,"projects/500942601540",null,[20],[2]]`:
				_ = json.NewEncoder(w).Encode([]any{
					[]any{
						[]any{
							[]any{
								[]any{
									[]any{"1776663651", 116580000},
									[]any{"15"},
								},
							},
							"gemini-2.5-flash",
						},
					},
					[]any{"1776660051", 116580000},
					[]any{"1776663651", 116580000},
					[]any{"60"},
				})
			case `[null,null,null,null,3,null,1,"projects/500942601540",2,[20],[1]]`:
				_ = json.NewEncoder(w).Encode([]any{
					[]any{
						[]any{
							[]any{
								[]any{
									[]any{"1776660051", 170950000},
									[]any{"3"},
								},
							},
							"gemini-2.5-flash",
						},
					},
					[]any{"1776660051", 170950000},
					[]any{"1776663651", 170950000},
					[]any{"3600"},
				})
			default:
				t.Fatalf("unexpected FetchMetricTimeSeries body = %s", string(body))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	previousBaseURL := googleAIStudioRPCBaseURL
	googleAIStudioRPCBaseURL = server.URL
	defer func() {
		googleAIStudioRPCBaseURL = previousBaseURL
	}()

	collector := &googleAIStudioObservabilityCollector{
		now: func() time.Time { return time.Unix(1718000000, 0) },
	}
	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		SchemaID:    "google",
		ProviderID: "account-google",
		SurfaceID:  "instance-google",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	for path, want := range map[string]string{
		"/ListCloudProjects":   `[]`,
		"/ListQuotaModels":     `[]`,
		"/ListModelRateLimits": `["projects/500942601540"]`,
	} {
		if got := requestBodies[path]; got != want {
			t.Fatalf("%s body = %q, want %q", path, got, want)
		}
	}
	if got, want := strings.Join(metricTimeSeriesBodies, "\n"), strings.Join([]string{
		`[null,null,null,null,3,null,2,"projects/500942601540",null,[20],[1]]`,
		`[null,null,null,null,3,null,2,"projects/500942601540",null,[20],[2]]`,
		`[null,null,null,null,3,null,1,"projects/500942601540",2,[20],[1]]`,
	}, "\n"); got != want {
		t.Fatalf("FetchMetricTimeSeries bodies = %q, want %q", got, want)
	}
	if got, want := len(result.GaugeRows), 11; got != want {
		t.Fatalf("metric rows = %d, want %d", got, want)
	}

	var foundTPD bool
	var foundTPDReset bool
	var foundRPMRemaining bool
	var foundTPMRemaining bool
	var foundRPDRemaining bool
	var foundUnknownQuotaType bool
	var foundNonTextCategoryModel bool
	for _, row := range result.GaugeRows {
		if row.Value <= 0 {
			t.Fatalf("unexpected non-positive quota metric row: %#v", row)
		}
		if row.Labels["model_category"] != googleAIStudioTextOutputModelCategory {
			t.Fatalf("model_category = %q, want %q", row.Labels["model_category"], googleAIStudioTextOutputModelCategory)
		}
		if row.Labels["model_id"] == "gemini-2.5-flash-tts" {
			foundNonTextCategoryModel = true
		}
		if row.Labels["model_id"] == "gemini-2.5-flash" &&
			row.Labels["quota_type"] == "TPD" &&
			row.Labels["resource"] == "tokens" &&
			row.Labels["window"] == "day" &&
			row.MetricName == googleAIStudioQuotaLimitMetric {
			foundTPD = true
		}
		if row.Labels["model_id"] == "gemini-2.5-flash" &&
			row.Labels["quota_type"] == "TPD" &&
			row.Labels["resource"] == "tokens" &&
			row.Labels["window"] == "day" &&
			row.MetricName == providerQuotaResetTimestampMetric {
			foundTPDReset = true
		}
		if row.Labels["model_id"] == "gemini-2.5-flash" &&
			row.Labels["quota_type"] == "RPM" &&
			row.MetricName == providerQuotaRemainingMetric &&
			row.Value == 2 {
			foundRPMRemaining = true
		}
		if row.Labels["model_id"] == "gemini-2.5-flash" &&
			row.Labels["quota_type"] == "TPM" &&
			row.MetricName == providerQuotaRemainingMetric &&
			row.Value == 249985 {
			foundTPMRemaining = true
		}
		if row.Labels["model_id"] == "gemini-2.5-flash" &&
			row.Labels["quota_type"] == "RPD" &&
			row.Labels["resource"] == "requests" &&
			row.Labels["window"] == "day" &&
			row.MetricName == providerQuotaRemainingMetric &&
			row.Value == 17 {
			foundRPDRemaining = true
		}
		if strings.HasPrefix(row.Labels["quota_type"], "TYPE_") {
			foundUnknownQuotaType = true
		}
	}
	if !foundTPD {
		t.Fatalf("expected gemini-2.5-flash TPD row")
	}
	if !foundTPDReset {
		t.Fatalf("expected gemini-2.5-flash TPD reset row")
	}
	if !foundRPMRemaining {
		t.Fatalf("expected gemini-2.5-flash RPM remaining row")
	}
	if !foundTPMRemaining {
		t.Fatalf("expected gemini-2.5-flash TPM remaining row")
	}
	if !foundRPDRemaining {
		t.Fatalf("expected gemini-2.5-flash RPD remaining row")
	}
	if foundUnknownQuotaType {
		t.Fatalf("unexpected TYPE_* quota_type row")
	}
	if foundNonTextCategoryModel {
		t.Fatalf("unexpected quota row from non-text model category")
	}
}

func TestGoogleAIStudioObservabilityCollectorCollectOmitsDirectSecretHeaders(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty before egress auth injection", got)
		}
		if got := r.Header.Get("Cookie"); got != "" {
			t.Fatalf("Cookie = %q, want empty before egress auth injection", got)
		}
		if got := r.Header.Get("X-Goog-Api-Key"); got != "" {
			t.Fatalf("X-Goog-Api-Key = %q, want empty before egress auth injection", got)
		}
		switch r.URL.Path {
		case "/ListCloudProjects":
			_ = json.NewEncoder(w).Encode([]any{
				[]any{
					[]any{"projects/946397203396", "gen-lang-client-123", "Default Gemini Project", []any{}, 1, 20},
				},
			})
		case "/ListQuotaModels":
			_ = json.NewEncoder(w).Encode([]any{
				[]any{
					[]any{"gemini-2.5-flash", nil, 4, "gemini-2.5", []any{[]any{"models/gemini-2.5-flash", []any{1, 2, 3}, []any{8, 10}}}, "gemini-2.5-flash"},
				},
			})
		case "/ListModelRateLimits":
			_ = json.NewEncoder(w).Encode([]any{
				[]any{
					[]any{"gemini-2.5-flash", 20, 1, 1, []any{"5"}, 4},
				},
			})
		case "/FetchMetricTimeSeries":
			_ = json.NewEncoder(w).Encode([]any{
				nil,
				[]any{"1776660051", 0},
				[]any{"1776663651", 0},
				[]any{"60"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	previousBaseURL := googleAIStudioRPCBaseURL
	googleAIStudioRPCBaseURL = server.URL
	defer func() {
		googleAIStudioRPCBaseURL = previousBaseURL
	}()

	collector := &googleAIStudioObservabilityCollector{
		now: func() time.Time { return time.Unix(1718000000, 0) },
	}
	result, err := collector.Collect(context.Background(), ObservabilityCollectInput{
		SchemaID:    "google",
		ProviderID: "account-google",
		SurfaceID:  "instance-google",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if got := requestCount; got == 0 {
		t.Fatalf("request count = 0")
	}
	if got := len(result.GaugeRows); got == 0 {
		t.Fatalf("metric rows = 0")
	}
}
