package providerservice

import (
	"context"
	"strings"
	"time"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	authv1 "code-code.internal/go-contract/platform/auth/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authCredentialService struct {
	client authv1.AuthServiceClient
}

func newAuthCredentialService(client authv1.AuthServiceClient) authCredentialService {
	return authCredentialService{client: client}
}

func (s authCredentialService) UpdateAPIKey(ctx context.Context, request providers.CredentialAPIKeyUpdate) (*managementv1.CredentialView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	response, err := s.client.UpdateAPIKeyCredential(ctx, &authv1.UpdateAPIKeyCredentialRequest{
		CredentialId: strings.TrimSpace(request.CredentialID),
		DisplayName:  strings.TrimSpace(request.DisplayName),
		Purpose:      purposeName(request.Purpose),
		VendorId:     strings.TrimSpace(request.VendorID),
		ApiKey:       strings.TrimSpace(request.APIKey),
	})
	if err != nil {
		return nil, err
	}
	return response.GetCredential(), nil
}

func (s authCredentialService) UpdateSession(ctx context.Context, request providers.CredentialSessionUpdate) (*managementv1.CredentialView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	response, err := s.client.UpdateSessionCredential(ctx, &authv1.UpdateSessionCredentialRequest{
		CredentialId: strings.TrimSpace(request.CredentialID),
		DisplayName:  strings.TrimSpace(request.DisplayName),
		Purpose:      purposeName(request.Purpose),
		VendorId:     strings.TrimSpace(request.VendorID),
		SchemaId:     strings.TrimSpace(request.SchemaID),
		RequiredKeys: append([]string(nil), request.RequiredKeys...),
		Values:       cloneStringMap(request.Values),
		MergeValues:  request.MergeValues,
	})
	if err != nil {
		return nil, err
	}
	return response.GetCredential(), nil
}

func (s authCredentialService) MergeCredentialMaterialValues(ctx context.Context, credentialID string, values map[string]string) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.client.MergeCredentialMaterialValues(ctx, &authv1.MergeCredentialMaterialValuesRequest{
		CredentialId: strings.TrimSpace(credentialID),
		Values:       cloneStringMap(values),
	})
	return err
}

func (s authCredentialService) ReadCredentialMaterialFields(
	ctx context.Context,
	credentialID string,
	policyRef *authv1.CredentialMaterialReadPolicyRef,
	fieldIDs []string,
) (map[string]string, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	response, err := s.client.ReadCredentialMaterialFields(ctx, &authv1.ReadCredentialMaterialFieldsRequest{
		CredentialId: strings.TrimSpace(credentialID),
		FieldIds:     append([]string(nil), fieldIDs...),
		PolicyRef:    policyRef,
	})
	if err != nil {
		return nil, err
	}
	return cloneStringMap(response.GetValues()), nil
}

func (s authCredentialService) Rename(ctx context.Context, credentialID, displayName string) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.client.RenameCredential(ctx, &authv1.RenameCredentialRequest{
		CredentialId: strings.TrimSpace(credentialID),
		DisplayName:  strings.TrimSpace(displayName),
	})
	return err
}

func (s authCredentialService) Delete(ctx context.Context, credentialID string) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.client.DeleteCredential(ctx, &authv1.DeleteCredentialRequest{CredentialId: strings.TrimSpace(credentialID)})
	return err
}

func (s authCredentialService) Exists(ctx context.Context, credentialID string) (bool, error) {
	if err := s.ready(); err != nil {
		return false, err
	}
	response, err := s.client.ListCredentials(ctx, &authv1.ListCredentialsRequest{})
	if err != nil {
		return false, err
	}
	credentialID = strings.TrimSpace(credentialID)
	for _, item := range response.GetItems() {
		if strings.TrimSpace(item.GetCredentialId()) == credentialID {
			return true, nil
		}
	}
	return false, nil
}

func (s authCredentialService) CredentialSubjectSummary(ctx context.Context, credentialID string) ([]*managementv1.CredentialSubjectSummaryFieldView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	response, err := s.client.GetCredentialSubjectSummary(ctx, &authv1.GetCredentialSubjectSummaryRequest{
		CredentialId: strings.TrimSpace(credentialID),
	})
	if err != nil {
		return nil, err
	}
	return response.GetFields(), nil
}

func (s authCredentialService) EnsureFresh(ctx context.Context, credentialID string, minTTL time.Duration) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.client.EnsureFresh(ctx, &authv1.EnsureFreshRequest{
		CredentialId:  strings.TrimSpace(credentialID),
		MinTtlSeconds: int64(minTTL / time.Second),
	})
	return err
}

func (s authCredentialService) RuntimeProjection(ctx context.Context, credentialID string) (*authv1.CredentialRuntimeProjection, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	response, err := s.client.GetCredentialRuntimeProjection(ctx, &authv1.GetCredentialRuntimeProjectionRequest{
		CredentialId: strings.TrimSpace(credentialID),
	})
	if err != nil {
		return nil, err
	}
	return response.GetCredential(), nil
}

func (s authCredentialService) CreateAPIKey(ctx context.Context, request providerconnect.CredentialAPIKeyCreate) (string, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	response, err := s.client.CreateAPIKeyCredential(ctx, &authv1.CreateAPIKeyCredentialRequest{
		CredentialId: strings.TrimSpace(request.CredentialID),
		DisplayName:  strings.TrimSpace(request.DisplayName),
		Purpose:      purposeName(credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_DATA_PLANE),
		VendorId:     strings.TrimSpace(request.VendorID),
		ApiKey:       strings.TrimSpace(request.APIKey),
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response.GetCredential().GetCredentialId()), nil
}

func (s authCredentialService) CreateSession(ctx context.Context, request providerconnect.CredentialSessionCreate) (string, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	response, err := s.client.CreateSessionCredential(ctx, &authv1.CreateSessionCredentialRequest{
		CredentialId: strings.TrimSpace(request.CredentialID),
		DisplayName:  strings.TrimSpace(request.DisplayName),
		Purpose:      purposeName(credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_MANAGEMENT_PLANE),
		VendorId:     strings.TrimSpace(request.VendorID),
		SchemaId:     strings.TrimSpace(request.SchemaID),
		RequiredKeys: append([]string(nil), request.RequiredKeys...),
		Values:       cloneStringMap(request.Values),
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response.GetCredential().GetCredentialId()), nil
}

func (s authCredentialService) ready() error {
	if s.client == nil {
		return status.Error(codes.Unavailable, "auth service is unavailable")
	}
	return nil
}

func purposeName(purpose credentialv1.CredentialPurpose) string {
	if purpose == credentialv1.CredentialPurpose_CREDENTIAL_PURPOSE_UNSPECIFIED {
		return ""
	}
	return purpose.String()
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
