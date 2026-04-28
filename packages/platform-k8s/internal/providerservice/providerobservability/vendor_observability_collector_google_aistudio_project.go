package providerobservability

import (
	"fmt"
	"strconv"
	"strings"
)

type googleAIStudioProject struct {
	Path            string
	ClientProjectID string
	DisplayName     string
	Tier            string
	TierCode        int
}

func resolveGoogleAIStudioProject(payload []any, projectID string) (googleAIStudioProject, error) {
	rows, err := googleAIStudioPayloadRows(payload)
	if err != nil {
		return googleAIStudioProject{}, err
	}
	target := strings.TrimSpace(projectID)
	if target == "" {
		return googleAIStudioProject{}, fmt.Errorf("project_id is empty")
	}
	for _, item := range rows {
		row, ok := googleAIStudioPayloadRow(item)
		if !ok {
			continue
		}
		project := googleAIStudioProject{
			Path:            googleAIStudioStringAt(row, 0),
			ClientProjectID: googleAIStudioStringAt(row, 1),
			DisplayName:     googleAIStudioStringAt(row, 2),
		}
		project.TierCode, _ = googleAIStudioIntAt(row, 5)
		project.Tier = googleAIStudioTierName(project.TierCode)
		if googleAIStudioProjectMatches(project, target) {
			return project, nil
		}
	}
	return googleAIStudioProject{}, fmt.Errorf("project %q not found in ListCloudProjects", target)
}

func normalizeGoogleAIStudioProjectPath(projectID string) (string, bool) {
	target := strings.TrimSpace(projectID)
	if target == "" {
		return "", false
	}
	if strings.HasPrefix(target, "projects/") {
		projectNumber := strings.TrimSpace(strings.TrimPrefix(target, "projects/"))
		if projectNumber == "" {
			return "", false
		}
		return "projects/" + projectNumber, true
	}
	if !googleAIStudioDigitsOnly(target) {
		return "", false
	}
	return "projects/" + target, true
}

func googleAIStudioDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func googleAIStudioProjectMatches(project googleAIStudioProject, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	switch {
	case project.Path == target:
		return true
	case strings.TrimPrefix(project.Path, "projects/") == target:
		return true
	case project.ClientProjectID == target:
		return true
	default:
		return false
	}
}

func googleAIStudioTierName(code int) string {
	switch code {
	case 20:
		return "FREE"
	case 30:
		return "TIER_1"
	case 40:
		return "TIER_2"
	case 50:
		return "TIER_3"
	default:
		if code <= 0 {
			return ""
		}
		return "TIER_" + strconv.Itoa(code)
	}
}
