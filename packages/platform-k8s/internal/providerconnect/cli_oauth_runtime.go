package providerconnect

func (r providerConnectRuntime) sessionViewRuntime() providerConnectSessionViewRuntime {
	return newProviderConnectSessionViewRuntime(r.queries)
}

func (r providerConnectRuntime) sessionStartRuntime() providerConnectSessionStartRuntime {
	return newProviderConnectSessionStartRuntime(r.sessions, r.sessionViewRuntime())
}

func (r providerConnectRuntime) apiKeyResolutionRuntime() providerConnectAPIKeyResolutionRuntime {
	return newProviderConnectAPIKeyResolutionRuntime(r.support, r.queries)
}

func (r providerConnectRuntime) cliOAuthResolutionRuntime() providerConnectCLIOAuthResolutionRuntime {
	return newProviderConnectCLIOAuthResolutionRuntime(r.support, r.queries)
}

func (r providerConnectRuntime) oauthFinalizeRuntime() providerConnectOAuthFinalizeRuntime {
	return newProviderConnectOAuthFinalizeRuntime(
		r.resources,
		r.queries,
		r.postConnect,
		r.logger,
	)
}

func (r providerConnectRuntime) sessionSyncRuntime() providerConnectSessionSyncRuntime {
	return newProviderConnectSessionSyncRuntime(r.sessions.oauth, r.oauthFinalizeRuntime())
}

func (r providerConnectRuntime) sessionQueryRuntime() providerConnectSessionQueryRuntime {
	return newProviderConnectSessionQueryRuntime(
		r.sessions.store,
		r.sessionSyncRuntime(),
		r.sessionViewRuntime(),
	)
}
