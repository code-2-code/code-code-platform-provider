package providerobservability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"code-code.internal/platform-k8s/internal/egressauth"
)

const (
	googleAIStudioQuotaLimitMetric = providerQuotaLimitMetric

	googleAIStudioCollectorID = "google-aistudio-quotas"

	googleAIStudioDefaultOrigin   = "https://aistudio.google.com"
	googleAIStudioDefaultAuthUser = "0"
)

var googleAIStudioRPCBaseURL = "https://alkalimakersuite-pa.clients6.google.com/$rpc/google.internal.alkali.applications.makersuite.v1.MakerSuiteService"

func init() {
	registerVendorObservabilityCollectorFactory(googleAIStudioCollectorID, NewGoogleAIStudioVendorObservabilityCollector)
}

func NewGoogleAIStudioVendorObservabilityCollector() VendorObservabilityCollector {
	return &googleAIStudioVendorObservabilityCollector{now: time.Now}
}

type googleAIStudioVendorObservabilityCollector struct {
	now func() time.Time
}

func (c *googleAIStudioVendorObservabilityCollector) CollectorID() string {
	return googleAIStudioCollectorID
}

func (c *googleAIStudioVendorObservabilityCollector) AuthAdapterID() string {
	return egressauth.AuthAdapterGoogleAIStudioSessionID
}

func (c *googleAIStudioVendorObservabilityCollector) Collect(ctx context.Context, input VendorObservabilityCollectInput) (result *VendorObservabilityCollectResult, err error) {
	ctx, span := startVendorObservabilityCollectSpan(ctx, c.CollectorID())
	defer func() {
		finishVendorObservabilityCollectSpan(span, err)
		span.End()
	}()
	if input.HTTPClient == nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: http client is nil")
	}
	session := input.ObservabilityCredential
	cookieHeader := observabilitySessionValue(session, "cookie")
	pageAPIKey := observabilitySessionValue(session, "page_api_key", "pageApiKey")
	projectID := observabilitySessionValue(session, "project_id", "projectId")
	requestAuthorization := googleAIStudioRequestAuthorizationHeader(observabilitySessionValue(session, "authorization"))
	authUser := observabilitySessionValue(session, "authuser", "auth_user", "x_goog_authuser")
	if authUser == "" {
		authUser = googleAIStudioDefaultAuthUser
	}
	recordVendorObservabilityCredentialPresence(span, "request_cookie", cookieHeader != "")
	recordVendorObservabilityCredentialPresence(span, "page_api_key", pageAPIKey != "")
	recordVendorObservabilityCredentialPresence(span, "project_id", projectID != "")
	recordVendorObservabilityCredentialPresence(span, "request_authorization", requestAuthorization != "")
	origin := observabilitySessionValue(session, "origin")
	if origin == "" {
		origin = googleAIStudioDefaultOrigin
	}
	if cookieHeader == "" || pageAPIKey == "" || projectID == "" {
		return nil, unauthorizedVendorObservabilityError("google ai studio quotas: cookie, page_api_key, and project_id are required")
	}
	now := c.now().UTC()
	authHeader := requestAuthorization
	if authHeader == "" {
		if strings.Contains(cookieHeader, egressauth.Placeholder) {
			authHeader = egressauth.Placeholder
		} else {
			authHeader, err = googleAIStudioAuthorizationHeader(cookieHeader, origin, now)
			if err != nil {
				return nil, unauthorizedVendorObservabilityError(err.Error())
			}
		}
	}

	var projectPath string
	tierHint := googleAIStudioTierHint{}
	if path, ok := normalizeGoogleAIStudioProjectPath(projectID); ok {
		projectPath = path
	} else {
		cloudProjectsBody, callErr := c.call(ctx, input.HTTPClient, googleAIStudioRPCCallInput{
			Method:        "ListCloudProjects",
			Authorization: authHeader,
			AuthUser:      authUser,
			PageAPIKey:    pageAPIKey,
			CookieHeader:  cookieHeader,
			Origin:        origin,
		})
		if callErr != nil {
			return nil, callErr
		}
		cloudProjects, decodeErr := decodeGoogleAIStudioRPCBody(cloudProjectsBody)
		if decodeErr != nil {
			return nil, fmt.Errorf("providerobservability: google ai studio quotas: decode ListCloudProjects: %w", decodeErr)
		}
		project, resolveErr := resolveGoogleAIStudioProject(cloudProjects, projectID)
		if resolveErr != nil {
			return nil, fmt.Errorf("providerobservability: google ai studio quotas: resolve project %q: %w", projectID, resolveErr)
		}
		projectPath = project.Path
		tierHint = googleAIStudioTierHint{
			TierCode: project.TierCode,
		}
	}

	rateLimitsBody, err := c.call(ctx, input.HTTPClient, googleAIStudioRPCCallInput{
		Method:        "ListModelRateLimits",
		Authorization: authHeader,
		AuthUser:      authUser,
		PageAPIKey:    pageAPIKey,
		CookieHeader:  cookieHeader,
		Origin:        origin,
		ProjectPath:   projectPath,
	})
	if err != nil {
		return nil, err
	}
	rateLimits, err := decodeGoogleAIStudioRPCBody(rateLimitsBody)
	if err != nil {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: decode ListModelRateLimits: %w", err)
	}

	quotaModelsBody, err := c.call(ctx, input.HTTPClient, googleAIStudioRPCCallInput{
		Method:        "ListQuotaModels",
		Authorization: authHeader,
		AuthUser:      authUser,
		PageAPIKey:    pageAPIKey,
		CookieHeader:  cookieHeader,
		Origin:        origin,
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
		Authorization: authHeader,
		AuthUser:      authUser,
		PageAPIKey:    pageAPIKey,
		CookieHeader:  cookieHeader,
		Origin:        origin,
		ProjectPath:   projectPath,
	}
	models, err = c.enrichGoogleAIStudioMetricTimeSeriesRows(ctx, input.HTTPClient, metricTimeSeriesInput, models)
	if err != nil {
		if isVendorObservabilityUnauthorizedError(err) {
			return nil, err
		}
	}
	rows := googleAIStudioMetricRows(models, now)
	if len(rows) == 0 {
		return nil, fmt.Errorf("providerobservability: google ai studio quotas: no quota data collected")
	}
	return &VendorObservabilityCollectResult{
		GaugeRows: rows,
	}, nil
}
