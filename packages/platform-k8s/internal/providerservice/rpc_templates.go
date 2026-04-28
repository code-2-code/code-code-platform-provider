package providerservice

import (
	"context"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

func (s *Server) ListTemplates(_ context.Context, _ *providerservicev1.ListTemplatesRequest) (*providerservicev1.ListTemplatesResponse, error) {
	return &providerservicev1.ListTemplatesResponse{Items: templateViewsToService(s.templates.List())}, nil
}

func (s *Server) ApplyTemplate(ctx context.Context, request *providerservicev1.ApplyTemplateRequest) (*providerservicev1.ApplyTemplateResponse, error) {
	result, err := s.templates.Apply(ctx, applyTemplateRequestToManagement(request))
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ApplyTemplateResponse{Result: applyTemplateResultToService(result)}, nil
}
