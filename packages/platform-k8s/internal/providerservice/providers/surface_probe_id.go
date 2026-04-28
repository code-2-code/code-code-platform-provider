package providers

import "strings"

func SurfaceModelCatalogProbeID(surfaceID string) string {
	surfaceID = strings.TrimSpace(surfaceID)
	if surfaceID == "" {
		return ""
	}
	return "surface." + surfaceID
}
