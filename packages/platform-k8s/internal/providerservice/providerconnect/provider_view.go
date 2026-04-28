package providerconnect

// CredentialSubjectSummaryFieldView is the providerconnect-owned credential subject summary row.
type CredentialSubjectSummaryFieldView struct {
	FieldID string
	Label   string
	Value   string
}

func (v *CredentialSubjectSummaryFieldView) GetFieldId() string {
	if v == nil {
		return ""
	}
	return v.FieldID
}

func (v *CredentialSubjectSummaryFieldView) GetLabel() string {
	if v == nil {
		return ""
	}
	return v.Label
}

func (v *CredentialSubjectSummaryFieldView) GetValue() string {
	if v == nil {
		return ""
	}
	return v.Value
}

// ProviderView is the providerconnect-owned provider projection.
type ProviderView struct {
	ProviderID               string
	DisplayName              string
	VendorID                 string
	ProviderCredentialID     string
	Surfaces                 []*ProviderSurfaceBindingView
	IconURL                  string
	CredentialSubjectSummary []*CredentialSubjectSummaryFieldView
}

func (v *ProviderView) GetProviderId() string {
	if v == nil {
		return ""
	}
	return v.ProviderID
}

func (v *ProviderView) GetDisplayName() string {
	if v == nil {
		return ""
	}
	return v.DisplayName
}

func (v *ProviderView) GetVendorId() string {
	if v == nil {
		return ""
	}
	return v.VendorID
}

func (v *ProviderView) GetProviderCredentialId() string {
	if v == nil {
		return ""
	}
	return v.ProviderCredentialID
}

func (v *ProviderView) GetSurfaces() []*ProviderSurfaceBindingView {
	if v == nil || len(v.Surfaces) == 0 {
		return nil
	}
	return append([]*ProviderSurfaceBindingView(nil), v.Surfaces...)
}

func (v *ProviderView) GetIconUrl() string {
	if v == nil {
		return ""
	}
	return v.IconURL
}

func (v *ProviderView) GetCredentialSubjectSummary() []*CredentialSubjectSummaryFieldView {
	if v == nil || len(v.CredentialSubjectSummary) == 0 {
		return nil
	}
	return append([]*CredentialSubjectSummaryFieldView(nil), v.CredentialSubjectSummary...)
}
