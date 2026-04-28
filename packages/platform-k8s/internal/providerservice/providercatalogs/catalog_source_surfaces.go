package providercatalogs

import (
	credentialv1 "code-code.internal/go-contract/credential/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	surfaceregistry "code-code.internal/platform-k8s/internal/supportservice/providersurfaces/registry"
)

func registerSurfaceSources(registry *catalogSourceRegistry) error {
	for _, surface := range surfaceregistry.List() {
		if !supportsBestEffortCatalogProbe(surface) {
			continue
		}
		if err := registry.register(&surfaceCatalogSource{
			sourceRef: newCatalogSourceRef(providers.SurfaceModelCatalogProbeID(surface.GetSurfaceId())),
		}); err != nil {
			return err
		}
	}
	return nil
}

type surfaceCatalogSource struct {
	sourceRef catalogSourceRef
}

func (s *surfaceCatalogSource) ref() catalogSourceRef {
	return s.sourceRef
}

func supportsBestEffortCatalogProbe(surface *providerv1.ProviderSurface) bool {
	if surface == nil {
		return false
	}
	return surface.GetKind() == providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API &&
		supportsCredentialKind(surface.GetSupportedCredentialKinds(), credentialv1.CredentialKind_CREDENTIAL_KIND_API_KEY) &&
		(surfaceModelCatalogProbeMethod(surface) == providerv1.ProviderSurfaceModelCatalogProbeMethod_PROVIDER_SURFACE_MODEL_CATALOG_PROBE_METHOD_PROTOCOL_BEST_EFFORT ||
			surfaceModelCatalogProbeMethod(surface) == providerv1.ProviderSurfaceModelCatalogProbeMethod_PROVIDER_SURFACE_MODEL_CATALOG_PROBE_METHOD_STATIC ||
			surfaceModelCatalogProbeMethod(surface) == providerv1.ProviderSurfaceModelCatalogProbeMethod_PROVIDER_SURFACE_MODEL_CATALOG_PROBE_METHOD_ADAPTER)
}

func supportsCredentialKind(values []credentialv1.CredentialKind, want credentialv1.CredentialKind) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func surfaceModelCatalogProbeMethod(surface *providerv1.ProviderSurface) providerv1.ProviderSurfaceModelCatalogProbeMethod {
	if surface == nil || surface.GetProbes() == nil || surface.GetProbes().GetModelCatalog() == nil {
		return providerv1.ProviderSurfaceModelCatalogProbeMethod_PROVIDER_SURFACE_MODEL_CATALOG_PROBE_METHOD_UNSPECIFIED
	}
	return surface.GetProbes().GetModelCatalog().GetMethod()
}
