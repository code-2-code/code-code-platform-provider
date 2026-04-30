package providerobservability

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	googleAIStudioQuotaLimitMetric = providerQuotaLimitMetric

	googleAIStudioCollectorID = "google-aistudio-quotas"

	googleAIStudioDefaultOrigin   = "https://aistudio.google.com"
	googleAIStudioRequestAuthUser = "0"

	googleAIStudioRPCHost       = "alkalimakersuite-pa.clients6.google.com"
	googleAIStudioRPCPathPrefix = "/$rpc/google.internal.alkali.applications.makersuite.v1.MakerSuiteService"
)

var googleAIStudioRPCBaseURL = "https://" + googleAIStudioRPCHost + googleAIStudioRPCPathPrefix

func init() {
	registerVendorCollectorFactory(googleAIStudioCollectorID, NewGoogleAIStudioObservabilityCollector)
}

func NewGoogleAIStudioObservabilityCollector() ObservabilityCollector {
	return &googleAIStudioObservabilityCollector{now: time.Now}
}

type googleAIStudioObservabilityCollector struct {
	now func() time.Time
}

func (c *googleAIStudioObservabilityCollector) CollectorID() string {
	return googleAIStudioCollectorID
}

func (c *googleAIStudioObservabilityCollector) Collect(ctx context.Context, input ObservabilityCollectInput) (result *ObservabilityCollectResult, err error) {
	ctx, span := startSurfaceObservabilityCollectSpan(ctx, c.CollectorID())
	defer func() {
		finishSurfaceObservabilityCollectSpan(span, err)
		span.End()
	}()
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: http client is nil")
	}
	projectID := ""
	origin := googleAIStudioDefaultOrigin
	now := c.now().UTC()
	baseURL := strings.TrimSpace(googleAIStudioRPCBaseURL)

	var projectPath string
	tierHint := googleAIStudioTierHint{}
	if path, ok := normalizeGoogleAIStudioProjectPath(projectID); ok {
		projectPath = path
	} else {
		projectPath, tierHint, err = c.resolveProjectPath(ctx, input.HTTPClient, baseURL, origin, projectID)
		if err != nil {
			return nil, err
		}
	}

	rateLimitsBody, err := c.call(ctx, input.HTTPClient, googleAIStudioRPCCallInput{
		BaseURL:     baseURL,
		Method:      "ListModelRateLimits",
		Origin:      origin,
		ProjectPath: projectPath,
	})
	if err != nil {
		return nil, err
	}
	rateLimits, err := decodeGoogleAIStudioRPCBody(rateLimitsBody)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: decode ListModelRateLimits: %w", err)
	}

	quotaModelsBody, err := c.call(ctx, input.HTTPClient, googleAIStudioRPCCallInput{
		BaseURL: baseURL,
		Method:  "ListQuotaModels",
		Origin:  origin,
	})
	if err != nil {
		return nil, err
	}
	quotaModels, err := decodeGoogleAIStudioRPCBody(quotaModelsBody)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: decode ListQuotaModels: %w", err)
	}
	modelMeta, err := parseGoogleAIStudioQuotaModels(quotaModels)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: parse ListQuotaModels: %w", err)
	}
	models, err := parseGoogleAIStudioRateLimits(rateLimits, tierHint, modelMeta)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: parse ListModelRateLimits: %w", err)
	}
	metricTimeSeriesInput := googleAIStudioRPCCallInput{
		BaseURL:     baseURL,
		Origin:      origin,
		ProjectPath: projectPath,
	}
	models, err = c.enrichGoogleAIStudioMetricTimeSeriesRows(ctx, input.HTTPClient, metricTimeSeriesInput, models)
	if err != nil {
		if isObservabilityUnauthorizedError(err) {
			return nil, err
		}
	}
	rows := googleAIStudioMetricRows(models, now)
	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: no quota data collected")
	}
	return &ObservabilityCollectResult{
		GaugeRows: rows,
	}, nil
}

func (c *googleAIStudioObservabilityCollector) resolveProjectPath(
	ctx context.Context,
	httpClient *http.Client,
	baseURL string,
	origin string,
	projectID string,
) (string, googleAIStudioTierHint, error) {
	cloudProjectsBody, callErr := c.call(ctx, httpClient, googleAIStudioRPCCallInput{
		BaseURL: baseURL,
		Method:  "ListCloudProjects",
		Origin:  origin,
	})
	if callErr != nil {
		return "", googleAIStudioTierHint{}, callErr
	}
	cloudProjects, decodeErr := decodeGoogleAIStudioRPCBody(cloudProjectsBody)
	if decodeErr != nil {
		return "", googleAIStudioTierHint{}, fmt.Errorf("providerobservability: google ai studio quotas: decode ListCloudProjects: %w", decodeErr)
	}
	project, resolveErr := resolveGoogleAIStudioProject(cloudProjects, projectID)
	if resolveErr != nil {
		return "", googleAIStudioTierHint{}, fmt.Errorf("providerobservability: google ai studio quotas: resolve project %q: %w", projectID, resolveErr)
	}
	return project.Path, googleAIStudioTierHint{TierCode: project.TierCode}, nil
}
