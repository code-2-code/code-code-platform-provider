package providerconnect

func (c *connectProviderCandidate) CustomAPIKeyTarget(displayName string) (*connectTarget, error) {
	return newConnectTarget(
		AddMethodAPIKey,
		displayName,
		"",
		"",
		c.SurfaceID(),
		c.Runtime(),
		"custom",
	)
}

func (c *connectProviderCandidate) VendorAPIKeyTarget(displayName, vendorID string) (*connectTarget, error) {
	return newConnectTarget(
		AddMethodAPIKey,
		displayName,
		vendorID,
		"",
		c.SurfaceID(),
		c.Runtime(),
		vendorID,
	)
}

func (c *connectProviderCandidate) CLIOAuthTarget(displayName, vendorID, cliID string) (*connectTarget, error) {
	return newConnectTarget(
		AddMethodCLIOAuth,
		displayName,
		vendorID,
		cliID,
		c.SurfaceID(),
		c.Runtime(),
		cliID,
	)
}
