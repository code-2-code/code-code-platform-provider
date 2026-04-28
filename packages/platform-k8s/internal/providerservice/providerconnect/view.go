package providerconnect

// SessionView is the providerconnect-owned session projection.
type SessionView struct {
	SessionID        string
	OAuthSessionID   string
	Phase            SessionPhase
	DisplayName      string
	AuthorizationURL string
	UserCode         string
	Message          string
	ErrorMessage     string
	AddMethod        AddMethod
	VendorID         string
	CLIID            string
	Provider         *ProviderView
}

func (v *SessionView) GetSessionId() string {
	if v == nil {
		return ""
	}
	return v.SessionID
}

func (v *SessionView) GetOauthSessionId() string {
	if v == nil {
		return ""
	}
	return v.OAuthSessionID
}

func (v *SessionView) GetPhase() SessionPhase {
	if v == nil {
		return SessionPhaseUnspecified
	}
	return v.Phase
}

func (v *SessionView) GetDisplayName() string {
	if v == nil {
		return ""
	}
	return v.DisplayName
}

func (v *SessionView) GetAuthorizationUrl() string {
	if v == nil {
		return ""
	}
	return v.AuthorizationURL
}

func (v *SessionView) GetUserCode() string {
	if v == nil {
		return ""
	}
	return v.UserCode
}

func (v *SessionView) GetMessage() string {
	if v == nil {
		return ""
	}
	return v.Message
}

func (v *SessionView) GetErrorMessage() string {
	if v == nil {
		return ""
	}
	return v.ErrorMessage
}

func (v *SessionView) GetAddMethod() AddMethod {
	if v == nil {
		return AddMethodUnspecified
	}
	return v.AddMethod
}

func (v *SessionView) GetVendorId() string {
	if v == nil {
		return ""
	}
	return v.VendorID
}

func (v *SessionView) GetCliId() string {
	if v == nil {
		return ""
	}
	return v.CLIID
}

func (v *SessionView) GetProvider() *ProviderView {
	if v == nil {
		return nil
	}
	return v.Provider
}

// ConnectResult is the providerconnect-owned connect outcome.
type ConnectResult struct {
	Provider *ProviderView
	Session  *SessionView
}

func (r *ConnectResult) GetProvider() *ProviderView {
	if r == nil {
		return nil
	}
	return r.Provider
}

func (r *ConnectResult) GetSession() *SessionView {
	if r == nil {
		return nil
	}
	return r.Session
}
