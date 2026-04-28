package providerobservability

import (
	"context"
	"strings"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
)

func (r *VendorObservabilityRunner) readCredentialMaterialFields(
	ctx context.Context,
	credentialID string,
	policyRef *authv1.CredentialMaterialReadPolicyRef,
	keys []string,
) (map[string]string, error) {
	credentialID = strings.TrimSpace(credentialID)
	if r == nil || r.credentialReader == nil || credentialID == "" || len(keys) == 0 {
		return nil, nil
	}
	values, err := r.credentialReader.ReadCredentialMaterialFields(ctx, credentialID, policyRef, keys)
	if err != nil {
		return nil, err
	}
	return trimStringMap(values), nil
}
