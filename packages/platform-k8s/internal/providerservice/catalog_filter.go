package providerservice

import (
	"regexp"
	"strings"

	"code-code.internal/platform-k8s/internal/providerservice/providercatalogs"
)

var googleCatalogExcludedFamilyPattern = regexp.MustCompile(
	`(?i)(^|[^a-z0-9])` +
		`(aqa|image|imagen|banana|veo|lyria|embedding|embed|tts|computer|robotics|audio|live|research)` +
		`([^a-z0-9]|$)`,
)

func providerCatalogModelFilter(input providercatalogs.ModelIDFilterInput) bool {
	switch strings.TrimSpace(input.VendorID) {
	case "google":
		return !googleCatalogExcludedFamilyPattern.MatchString(input.ProviderModelID)
	default:
		return true
	}
}
