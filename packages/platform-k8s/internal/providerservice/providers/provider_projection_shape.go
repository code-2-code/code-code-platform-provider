package providers

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
