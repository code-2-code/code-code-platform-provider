package providerservice

import (
	"context"
	"fmt"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

func (s *Server) ListProviderSurfaces(ctx context.Context, _ *providerservicev1.ListProviderSurfacesRequest) (*providerservicev1.ListProviderSurfacesResponse, error) {
	if s == nil || s.surfaceMetadata == nil {
		return nil, grpcError(fmt.Errorf("platformk8s/providerservice: provider surface service is not initialized"))
	}
	items, err := s.surfaceMetadata.List(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ListProviderSurfacesResponse{Items: items}, nil
}

func (s *Server) ListProviders(ctx context.Context, _ *providerservicev1.ListProvidersRequest) (*providerservicev1.ListProvidersResponse, error) {
	if s == nil || s.providers == nil {
		return nil, grpcError(fmt.Errorf("platformk8s/providerservice: provider service is not initialized"))
	}
	items, err := s.providers.List(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ListProvidersResponse{Items: providerViewsToService(items)}, nil
}
