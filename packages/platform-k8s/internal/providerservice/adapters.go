package providerservice

import (
	"context"
	"strings"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	providerv1 "code-code.internal/go-contract/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
	"code-code.internal/platform-k8s/internal/providerservice/providersurfacebindings"
)

type providerConnectSurfaceBindingAdapter struct {
	source *providersurfacebindings.Service
}

func (a providerConnectSurfaceBindingAdapter) ListProviderSurfaceBindings(ctx context.Context) ([]*providerconnect.ProviderSurfaceBindingView, error) {
	items, err := a.source.ListProviderSurfaceBindings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*providerconnect.ProviderSurfaceBindingView, 0, len(items))
	for _, item := range items {
		if next := providerConnectSurfaceBindingFromTransport(item); next != nil {
			out = append(out, next)
		}
	}
	return out, nil
}

func (a providerConnectSurfaceBindingAdapter) CreateProvider(ctx context.Context, provider *providerv1.Provider) (*providerconnect.ProviderView, error) {
	view, err := a.source.CreateProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	return providerConnectProviderFromTransport(view), nil
}

type providerConnectProviderAdapter struct {
	source *providers.Service
}

func (a providerConnectProviderAdapter) Get(ctx context.Context, accountID string) (*providerconnect.ProviderView, error) {
	view, err := a.source.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return providerConnectProviderFromTransport(view), nil
}

type cliSupportIconReferenceService struct {
	source interface {
		List(context.Context) ([]*supportv1.CLI, error)
	}
}

func (s cliSupportIconReferenceService) List(ctx context.Context) ([]*managementv1.CLIDefinitionView, error) {
	items, err := s.source.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*managementv1.CLIDefinitionView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		cliID := strings.TrimSpace(item.GetCliId())
		if cliID == "" {
			continue
		}
		out = append(out, &managementv1.CLIDefinitionView{CliId: cliID, IconUrl: strings.TrimSpace(item.GetIconUrl())})
	}
	return out, nil
}
