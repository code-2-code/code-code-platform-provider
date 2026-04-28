package providers

import managementv1 "code-code.internal/go-contract/platform/management/v1"

func compareProviderProjections(left, right *ProviderProjection) int {
	switch {
	case left.DisplayName() < right.DisplayName():
		return -1
	case left.DisplayName() > right.DisplayName():
		return 1
	case left.ID() < right.ID():
		return -1
	case left.ID() > right.ID():
		return 1
	default:
		return 0
	}
}

func compareSurfaces(left, right *managementv1.ProviderSurfaceBindingView) int {
	switch {
	case left.GetDisplayName() < right.GetDisplayName():
		return -1
	case left.GetDisplayName() > right.GetDisplayName():
		return 1
	case left.GetSurfaceId() < right.GetSurfaceId():
		return -1
	case left.GetSurfaceId() > right.GetSurfaceId():
		return 1
	default:
		return 0
	}
}
