package providerconnect

import (
	"strings"

	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/providersurfaces"
	"google.golang.org/protobuf/proto"
)

type connectSurfaceMetadata struct {
	value *supportv1.Surface
}

func newConnectSurfaceMetadata(value *supportv1.Surface) (*connectSurfaceMetadata, error) {
	if value == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface is invalid")
	}
	if strings.TrimSpace(value.GetSurfaceId()) == "" {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface id is empty")
	}
	return &connectSurfaceMetadata{
		value: proto.Clone(value).(*supportv1.Surface),
	}, nil
}

func (m *connectSurfaceMetadata) SurfaceID() string {
	if m == nil || m.value == nil {
		return ""
	}
	return strings.TrimSpace(m.value.GetSurfaceId())
}

func (m *connectSurfaceMetadata) ValidateCandidate(
	candidate *connectProviderCandidate,
	credentialKind credentialv1.CredentialKind,
) error {
	_ = credentialKind
	if candidate == nil || candidate.Endpoint() == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: provider endpoint is required")
	}
	if m == nil || m.value == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface %q is invalid", candidate.SurfaceID())
	}
	candidateKey := providerv1.EndpointKey(candidate.Endpoint())
	for _, endpoint := range providersurfaces.Endpoints(m.value) {
		if providerv1.EndpointKey(endpoint) == candidateKey {
			return nil
		}
	}
	return domainerror.NewValidation("platformk8s/providerconnect: provider endpoint is not supported by surface %q", m.SurfaceID())
}
