package providersurfacebindings

import (
	"strings"

	"code-code.internal/go-contract/domainerror"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/platform/resourcemeta"
	"google.golang.org/protobuf/proto"
)

type SurfaceBinding struct {
	value      *providerv1.ProviderSurfaceBinding
	providerID string
}

func NewProviderSurfaceBindingForCreate(providerID string, input *providerv1.ProviderSurfaceBinding) (*SurfaceBinding, error) {
	if input == nil {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: provider surface binding is nil")
	}
	next := cloneProviderSurfaceBindingProto(input)
	normalizeProviderSurfaceBinding(next)
	if strings.TrimSpace(providerID) == "" {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: provider_id is required")
	}
	if strings.TrimSpace(next.GetSurfaceId()) == "" {
		surfaceID, err := resourcemeta.EnsureSurfaceID("", providerSurfaceBindingDisplayName(next), "")
		if err != nil {
			return nil, err
		}
		next.SurfaceId = surfaceID
	}
	surface, err := newProviderSurfaceBinding(next)
	if err != nil {
		return nil, err
	}
	surface.providerID = strings.TrimSpace(providerID)
	return surface, nil
}

func NewProviderSurfaceBindingForUpdate(current *SurfaceBinding, input *providerv1.ProviderSurfaceBinding) (*SurfaceBinding, error) {
	if current == nil {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: current provider surface binding is invalid")
	}
	next := cloneProviderSurfaceBindingProto(input)
	normalizeProviderSurfaceBinding(next)
	if next.GetSurfaceId() != "" && strings.TrimSpace(next.GetSurfaceId()) != current.SurfaceID() {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: path surface id %q does not match payload %q", current.SurfaceID(), next.GetSurfaceId())
	}
	next.SurfaceId = current.SurfaceID()
	if strings.TrimSpace(next.GetRuntime().GetDisplayName()) == "" && next.GetRuntime() != nil {
		next.Runtime.DisplayName = current.DisplayName()
	}
	updated, err := newProviderSurfaceBinding(next)
	if err != nil {
		return nil, err
	}
	if err := current.ValidateMutableUpdate(updated); err != nil {
		return nil, err
	}
	updated.providerID = current.ProviderID()
	return updated, nil
}

func (s *SurfaceBinding) Proto() *providerv1.ProviderSurfaceBinding {
	if s == nil || s.value == nil {
		return nil
	}
	return proto.Clone(s.value).(*providerv1.ProviderSurfaceBinding)
}

func (s *SurfaceBinding) SurfaceID() string {
	if s == nil || s.value == nil {
		return ""
	}
	return strings.TrimSpace(s.value.GetSurfaceId())
}

func (s *SurfaceBinding) DisplayName() string {
	if s == nil || s.value == nil || s.value.GetRuntime() == nil {
		return ""
	}
	displayName := strings.TrimSpace(s.value.GetRuntime().GetDisplayName())
	if displayName != "" {
		return displayName
	}
	return s.SurfaceID()
}

func (s *SurfaceBinding) ProviderID() string {
	if s == nil || s.value == nil {
		return ""
	}
	return strings.TrimSpace(s.providerID)
}

func (s *SurfaceBinding) SurfaceKind() providerv1.ProviderSurfaceKind {
	if s == nil || s.value == nil || s.value.GetRuntime() == nil {
		return providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_UNSPECIFIED
	}
	if s.value.GetRuntime().GetApi() != nil {
		return providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_API
	}
	if s.value.GetRuntime().GetCli() != nil {
		return providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_CLI
	}
	return providerv1.ProviderSurfaceKind_PROVIDER_SURFACE_KIND_UNSPECIFIED
}

func (s *SurfaceBinding) CLIID() string {
	if s == nil || s.value == nil || s.value.GetRuntime() == nil || s.value.GetRuntime().GetCli() == nil {
		return ""
	}
	return strings.TrimSpace(s.value.GetRuntime().GetCli().GetCliId())
}

func (s *SurfaceBinding) ValidateMutableUpdate(next *SurfaceBinding) error {
	if s == nil || s.value == nil || next == nil || next.value == nil {
		return nil
	}
	if s.value.GetRuntime().GetOrigin() == providerv1.ProviderSurfaceOrigin_PROVIDER_SURFACE_ORIGIN_DERIVED &&
		!proto.Equal(s.value.GetRuntime(), next.value.GetRuntime()) {
		return domainerror.NewValidation("platformk8s/providersurfacebindings: derived provider surface binding %q is immutable", s.SurfaceID())
	}
	return nil
}

func newProviderSurfaceBinding(input *providerv1.ProviderSurfaceBinding) (*SurfaceBinding, error) {
	if input == nil {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: provider surface binding is nil")
	}
	next := cloneProviderSurfaceBindingProto(input)
	normalizeProviderSurfaceBinding(next)
	if err := providerv1.ValidateProviderSurfaceBinding(next); err != nil {
		return nil, domainerror.NewValidation("platformk8s/providersurfacebindings: invalid provider surface binding: %v", err)
	}
	return &SurfaceBinding{value: next}, nil
}
