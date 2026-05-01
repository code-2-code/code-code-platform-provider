package providerconnect

func (c *connectProviderCandidate) APIKeyTarget(displayName string) (*connectTarget, error) {
	target, err := newConnectTarget(
		AddMethodAPIKey,
		displayName,
		"",
		c.SurfaceID(),
		c.Models(),
		c.SurfaceID(),
	)
	if err != nil {
		return nil, err
	}
	target.CustomAPIKeySurface = c.CustomAPIKeySurface()
	return target, nil
}

func (c *connectProviderCandidate) CLIOAuthTarget(displayName, cliID string) (*connectTarget, error) {
	return newConnectTarget(
		AddMethodCLIOAuth,
		displayName,
		cliID,
		c.SurfaceID(),
		c.Models(),
		cliID,
	)
}
