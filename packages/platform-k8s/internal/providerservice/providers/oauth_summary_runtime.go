package providers

import (
	"context"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
)

type credentialSubjectSummaryRuntime struct {
	credentials credentialService
}

func newCredentialSubjectSummaryRuntime(credentials credentialService) credentialSubjectSummaryRuntime {
	return credentialSubjectSummaryRuntime{credentials: credentials}
}

func (r credentialSubjectSummaryRuntime) Apply(ctx context.Context, projections []*ProviderProjection) []*ProviderProjection {
	items := make([]*ProviderProjection, 0, len(projections))
	for _, projection := range projections {
		items = append(items, projection.WithCredentialSubjectSummary(r.resolve(ctx, projection)))
	}
	return items
}

func (r credentialSubjectSummaryRuntime) resolve(ctx context.Context, projection *ProviderProjection) []*managementv1.CredentialSubjectSummaryFieldView {
	if projection == nil || r.credentials == nil {
		return nil
	}
	fields, err := r.credentials.CredentialSubjectSummary(ctx, projection.SubjectSummaryCredentialID())
	if err != nil {
		return nil
	}
	return fields
}
