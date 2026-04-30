package providerservice

import (
	"testing"

	"code-code.internal/platform-k8s/internal/providerservice/providercatalogs"
)

func TestProviderCatalogModelFilterDropsTemporaryGoogleNonTextModels(t *testing.T) {
	t.Parallel()

	for _, modelID := range []string{
		"aqa",
		"aqa-model",
		"gemini-2.0-flash-preview-image-generation",
		"imagen-4.0-generate-preview",
		"gemini-2.5-flash-preview-tts",
		"gemini-2.5-computer-use-preview-10-2025",
		"gemini-robotics-er-1.5-preview",
		"deep-research-pro-preview-12-2025",
		"gemini-2.5-flash-native-audio-preview-09-2025",
		"gemini-3.1-flash-live-preview",
		"nano-banana",
		"text-embedding-004",
	} {
		modelID := modelID
		t.Run(modelID, func(t *testing.T) {
			t.Parallel()
			if providerCatalogModelFilter(providercatalogs.ModelIDFilterInput{
				VendorID:        "google",
				SurfaceID:       "gemini",
				ProviderModelID: modelID,
			}) {
				t.Fatalf("providerCatalogModelFilter(%q) = true, want false", modelID)
			}
		})
	}
}

func TestProviderCatalogModelFilterKeepsGoogleTextModels(t *testing.T) {
	t.Parallel()

	for _, modelID := range []string{
		"gemini-2.5-pro",
		"gemini-2.5-flash",
	} {
		modelID := modelID
		t.Run(modelID, func(t *testing.T) {
			t.Parallel()
			if !providerCatalogModelFilter(providercatalogs.ModelIDFilterInput{
				VendorID:        "google",
				SurfaceID:       "gemini",
				ProviderModelID: modelID,
			}) {
				t.Fatalf("providerCatalogModelFilter(%q) = false, want true", modelID)
			}
		})
	}
}

func TestProviderCatalogModelFilterDoesNotApplyGoogleRulesToOtherVendors(t *testing.T) {
	t.Parallel()

	if !providerCatalogModelFilter(providercatalogs.ModelIDFilterInput{
		VendorID:        "mistral",
		SurfaceID:       "openai-compatible",
		ProviderModelID: "image-test-model",
	}) {
		t.Fatal("providerCatalogModelFilter() applied google filter to non-google vendor")
	}
}
