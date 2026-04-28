package providerservice

import (
	"context"
	"strings"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
)

func (s *Server) UpdateProvider(ctx context.Context, request *providerservicev1.UpdateProviderRequest) (*providerservicev1.UpdateProviderResponse, error) {
	provider, err := s.providers.Update(ctx, request.GetProviderId(), providers.UpdateProviderCommand{DisplayName: request.GetDisplayName()})
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.UpdateProviderResponse{Provider: providerViewToService(provider)}, nil
}

func (s *Server) UpdateProviderAuthentication(ctx context.Context, request *providerservicev1.UpdateProviderAuthenticationRequest) (*providerservicev1.UpdateProviderAuthenticationResponse, error) {
	switch request.GetAuthMaterial().(type) {
	case *providerservicev1.UpdateProviderAuthenticationRequest_CliOauth:
		provider, err := s.providers.Get(ctx, request.GetProviderId())
		if err != nil {
			return nil, grpcError(err)
		}
		session, err := s.providerConnect.Reauthorize(ctx, providerConnectProviderFromTransport(provider))
		if err != nil {
			return nil, grpcError(err)
		}
		return &providerservicev1.UpdateProviderAuthenticationResponse{Outcome: &providerservicev1.UpdateProviderAuthenticationResponse_Session{Session: providerConnectSessionViewToTransport(session)}}, nil
	default:
		apiKey := request.GetApiKey()
		response, err := s.providers.UpdateAPIKeyAuthentication(ctx, request.GetProviderId(), providers.UpdateAPIKeyAuthenticationCommand{APIKey: apiKey.GetApiKey()})
		if err != nil {
			return nil, grpcError(err)
		}
		return updateProviderAuthenticationResponseToService(response), nil
	}
}

func (s *Server) UpdateProviderObservabilityAuthentication(ctx context.Context, request *providerservicev1.UpdateProviderObservabilityAuthenticationRequest) (*providerservicev1.UpdateProviderObservabilityAuthenticationResponse, error) {
	command, err := s.observabilityAuthenticationCommand(ctx, request.GetProviderId(), request.GetSessionMaterial())
	if err != nil {
		return nil, grpcError(err)
	}
	provider, err := s.providers.UpdateObservabilityAuthentication(ctx, request.GetProviderId(), command)
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.UpdateProviderObservabilityAuthenticationResponse{Provider: providerViewToService(provider)}, nil
}

func (s *Server) DeleteProvider(ctx context.Context, request *providerservicev1.DeleteProviderRequest) (*providerservicev1.DeleteProviderResponse, error) {
	if err := s.providers.Delete(ctx, request.GetProviderId()); err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.DeleteProviderResponse{Status: actionStatusOK}, nil
}

func (s *Server) ListProviderSurfaceBindings(ctx context.Context, _ *providerservicev1.ListProviderSurfaceBindingsRequest) (*providerservicev1.ListProviderSurfaceBindingsResponse, error) {
	items, err := s.providerSurfaceBindings.ListProviderSurfaceBindings(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ListProviderSurfaceBindingsResponse{Items: providerSurfaceBindingViewsToService(items)}, nil
}

func (s *Server) CreateProviderSurfaceBinding(ctx context.Context, request *providerservicev1.CreateProviderSurfaceBindingRequest) (*providerservicev1.CreateProviderSurfaceBindingResponse, error) {
	item, err := s.providerSurfaceBindings.CreateProviderSurfaceBinding(ctx, request.GetSurface().GetProviderId(), providerSurfaceBindingFromUpsertRequest(request.GetSurface()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.CreateProviderSurfaceBindingResponse{Surface: providerSurfaceBindingViewToService(item)}, nil
}

func (s *Server) UpdateProviderSurfaceBinding(ctx context.Context, request *providerservicev1.UpdateProviderSurfaceBindingRequest) (*providerservicev1.UpdateProviderSurfaceBindingResponse, error) {
	item, err := s.providerSurfaceBindings.UpdateProviderSurfaceBinding(ctx, request.GetSurfaceId(), providerSurfaceBindingFromUpsertRequest(request.GetSurface()))
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.UpdateProviderSurfaceBindingResponse{Surface: providerSurfaceBindingViewToService(item)}, nil
}

func (s *Server) DeleteProviderSurfaceBinding(ctx context.Context, request *providerservicev1.DeleteProviderSurfaceBindingRequest) (*providerservicev1.DeleteProviderSurfaceBindingResponse, error) {
	if err := s.providerSurfaceBindings.DeleteProviderSurfaceBinding(ctx, request.GetSurfaceId()); err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.DeleteProviderSurfaceBindingResponse{Status: actionStatusOK}, nil
}

func (s *Server) ConnectProvider(ctx context.Context, request *providerservicev1.ConnectProviderRequest) (*providerservicev1.ConnectProviderResponse, error) {
	command, err := providerConnectCommandFromRequest(request)
	if err != nil {
		return nil, grpcError(err)
	}
	result, err := s.providerConnect.Connect(ctx, command)
	if err != nil {
		return nil, grpcError(err)
	}
	return connectProviderResponseFromResult(result), nil
}

func (s *Server) GetProviderConnectSession(ctx context.Context, request *providerservicev1.GetProviderConnectSessionRequest) (*providerservicev1.GetProviderConnectSessionResponse, error) {
	session, err := s.providerConnect.GetSession(ctx, request.GetSessionId())
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.GetProviderConnectSessionResponse{Session: providerConnectSessionViewToTransport(session)}, nil
}

func (s *Server) ProbeProviderObservability(ctx context.Context, request *providerservicev1.ProbeProviderObservabilityRequest) (*providerservicev1.ProbeProviderObservabilityResponse, error) {
	providerIDs := normalizedProviderIDs(request.GetProviderIds())
	if len(providerIDs) == 0 && request.GetProviderId() != "" {
		providerIDs = []string{request.GetProviderId()}
	}
	if len(providerIDs) == 0 {
		ids, err := s.providerIDs(ctx)
		if err != nil {
			return nil, grpcError(err)
		}
		if len(ids) == 0 {
			return &providerservicev1.ProbeProviderObservabilityResponse{Message: "no providers to probe"}, nil
		}
		providerIDs = ids
	}
	if err := s.runProviderObservabilityProbe(ctx, providerIDs, providerObservabilityProbeTriggerFromTransport(request.GetTrigger())); err != nil {
		return nil, grpcError(err)
	}
	response := &providerservicev1.ProbeProviderObservabilityResponse{
		ProviderIds: providerIDs,
		Message:     "provider observability probe completed",
	}
	if len(providerIDs) == 1 {
		response.ProviderId = providerIDs[0]
	}
	return response, nil
}

func (s *Server) BindProviderCatalogs(ctx context.Context, _ *providerservicev1.BindProviderCatalogsRequest) (*providerservicev1.BindProviderCatalogsResponse, error) {
	if err := s.runProviderCatalogBinding(ctx); err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.BindProviderCatalogsResponse{Status: actionStatusOK}, nil
}

func (s *Server) WatchProviderStatusEvents(
	request *providerservicev1.WatchProviderStatusEventsRequest,
	stream providerservicev1.ProviderService_WatchProviderStatusEventsServer,
) error {
	return grpcError(s.StreamProviderStatusEvents(stream.Context(), request.GetProviderIds(), func(event *providerservicev1.ProviderStatusEvent) error {
		return stream.Send(&providerservicev1.WatchProviderStatusEventsResponse{Event: event})
	}))
}

func (s *Server) StreamProviderStatusEvents(
	ctx context.Context,
	providerIDs []string,
	yield func(*providerservicev1.ProviderStatusEvent) error,
) error {
	<-ctx.Done()
	return ctx.Err()
}

func normalizedProviderIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (s *Server) providerIDs(ctx context.Context) ([]string, error) {
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(providers))
	for _, provider := range providers {
		if provider == nil || strings.TrimSpace(provider.GetProviderId()) == "" {
			continue
		}
		ids = append(ids, strings.TrimSpace(provider.GetProviderId()))
	}
	return ids, nil
}

func providerObservabilityProbeTriggerFromTransport(trigger providerservicev1.ProviderObservabilityProbeTrigger) providerobservability.Trigger {
	switch trigger {
	case providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_CONNECT:
		return providerobservability.TriggerConnect
	case providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_SCHEDULE:
		return providerobservability.TriggerSchedule
	default:
		return providerobservability.TriggerManual
	}
}
