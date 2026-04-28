package providerconnect

import (
	"strings"

	apiprotocolv1 "code-code.internal/go-contract/api_protocol/v1"
	credentialv1 "code-code.internal/go-contract/credential/v1"
	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"google.golang.org/protobuf/proto"
)

type connectSurfaceMetadata struct {
	value *providerv1.ProviderSurface
}

func newConnectSurfaceMetadata(value *providerv1.ProviderSurface) (*connectSurfaceMetadata, error) {
	if value == nil {
		return nil, domainerror.NewValidation("platformk8s/providerconnect: provider surface is invalid")
	}
	if err := providerv1.ValidateProviderSurface(value); err != nil {
		return nil, domainerror.NewValidation(
			"platformk8s/providerconnect: invalid provider surface %q: %v",
			strings.TrimSpace(value.GetSurfaceId()),
			err,
		)
	}
	return &connectSurfaceMetadata{
		value: proto.Clone(value).(*providerv1.ProviderSurface),
	}, nil
}

func (m *connectSurfaceMetadata) SurfaceID() string {
	if m == nil || m.value == nil {
		return ""
	}
	return strings.TrimSpace(m.value.GetSurfaceId())
}

func (m *connectSurfaceMetadata) ValidateCandidate(
	candidate *connectSurfaceBindingCandidate,
	credentialKind credentialv1.CredentialKind,
) error {
	if candidate == nil || candidate.Runtime() == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface binding runtime is required")
	}
	if m == nil || m.value == nil {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface %q is invalid", candidate.SurfaceID())
	}
	if !surfaceSupportsCredentialKind(m.value.GetSupportedCredentialKinds(), credentialKind) {
		return domainerror.NewValidation(
			"platformk8s/providerconnect: provider surface %q does not support credential kind %s",
			m.SurfaceID(),
			credentialKind.String(),
		)
	}
	runtime := candidate.Runtime()
	if api := runtime.GetApi(); api != nil {
		if m.value.GetKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API {
			return domainerror.NewValidation("platformk8s/providerconnect: provider surface %q is not an API surface", m.SurfaceID())
		}
		if len(m.value.GetApi().GetSupportedProtocols()) > 0 &&
			!surfaceSupportsProtocol(m.value.GetApi().GetSupportedProtocols(), api.GetProtocol()) {
			return domainerror.NewValidation(
				"platformk8s/providerconnect: provider surface %q does not support protocol %s",
				m.SurfaceID(),
				api.GetProtocol().String(),
			)
		}
	}
	if runtime.GetCli() != nil && m.value.GetKind() != providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI {
		return domainerror.NewValidation("platformk8s/providerconnect: provider surface %q is not a CLI surface", m.SurfaceID())
	}
	return nil
}

func surfaceSupportsProtocol(values []apiprotocolv1.Protocol, want apiprotocolv1.Protocol) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func surfaceSupportsCredentialKind(values []credentialv1.CredentialKind, want credentialv1.CredentialKind) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
