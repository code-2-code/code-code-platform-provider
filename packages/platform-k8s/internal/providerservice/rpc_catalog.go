package providerservice

import (
	"context"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

func (s *Server) ListVendors(ctx context.Context, _ *providerservicev1.ListVendorsRequest) (*providerservicev1.ListVendorsResponse, error) {
	items, err := s.vendors.List(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ListVendorsResponse{Items: vendorViewsToService(items)}, nil
}

func (s *Server) ListCLIDefinitions(ctx context.Context, _ *providerservicev1.ListCLIDefinitionsRequest) (*providerservicev1.ListCLIDefinitionsResponse, error) {
	items, err := s.cliDefinitions.List(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ListCLIDefinitionsResponse{Items: cliDefinitionViewsToService(items)}, nil
}
