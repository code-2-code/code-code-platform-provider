package providerservice

import (
	"context"
	"strings"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/platform-k8s/internal/providerservice/providerobservability"
	"code-code.internal/platform-k8s/internal/providerservice/providers"
)

func (s *Server) UpdateProvider(ctx context.Context, request *providerservicev1.UpdateProviderRequest) (*providerservicev1.UpdateProviderResponse, error) {
	provider, err := s.providers.Update(ctx, request.GetProviderId(), providers.UpdateProviderCommand{DisplayName: request.GetProvider().GetDisplayName()})
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.UpdateProviderResponse{Provider: providerViewToService(provider)}, nil
}

func (s *Server) DeleteProvider(ctx context.Context, request *providerservicev1.DeleteProviderRequest) (*providerservicev1.DeleteProviderResponse, error) {
	if err := s.providers.Delete(ctx, request.GetProviderId()); err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.DeleteProviderResponse{Status: actionStatusOK}, nil
}

func (s *Server) ApplyProviderModelCatalog(ctx context.Context, request *providerservicev1.ApplyProviderModelCatalogRequest) (*providerservicev1.ApplyProviderModelCatalogResponse, error) {
	provider, err := s.providers.ApplyModelCatalog(ctx, request.GetProviderId(), request.GetModels())
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ApplyProviderModelCatalogResponse{Provider: providerViewToService(provider)}, nil
}

func (s *Server) ApplyProviderProbeStatus(ctx context.Context, request *providerservicev1.ApplyProviderProbeStatusRequest) (*providerservicev1.ApplyProviderProbeStatusResponse, error) {
	provider, err := s.providers.ApplyProbeStatus(ctx, request.GetProviderId(), request.GetProbeKind(), request.GetState())
	if err != nil {
		return nil, grpcError(err)
	}
	return &providerservicev1.ApplyProviderProbeStatusResponse{Provider: providerViewToService(provider)}, nil
}

func (s *Server) ProbeProviderObservability(ctx context.Context, request *providerservicev1.ProbeProviderObservabilityRequest) (*providerservicev1.ProbeProviderObservabilityResponse, error) {
	providerIDs, err := s.providerProbeIDs(ctx, request.GetProviderId(), request.GetProviderIds())
	if err != nil {
		return nil, grpcError(err)
	}
	if len(providerIDs) == 0 {
		return &providerservicev1.ProbeProviderObservabilityResponse{Message: "no providers to probe"}, nil
	}
	results, err := s.runProviderObservabilityProbe(ctx, providerIDs, providerObservabilityProbeTriggerFromTransport(request.GetTrigger()))
	if err != nil {
		return nil, grpcError(err)
	}
	response := &providerservicev1.ProbeProviderObservabilityResponse{
		ProviderIds: providerIDs,
		Message:     "provider observability probe completed",
	}
	if len(providerIDs) == 1 {
		response.ProviderId = providerIDs[0]
	}
	if len(results) == 1 && results[0] != nil {
		response.ProviderId = results[0].GetProviderId()
		response.Outcome = results[0].GetOutcome()
		response.Message = results[0].GetMessage()
		response.NextAllowedAt = results[0].GetNextAllowedAt()
		response.LastAttemptAt = results[0].GetLastAttemptAt()
	}
	return response, nil
}

func (s *Server) ProbeProviderModelCatalog(ctx context.Context, request *providerservicev1.ProbeProviderModelCatalogRequest) (*providerservicev1.ProbeProviderModelCatalogResponse, error) {
	providerIDs, err := s.providerProbeIDs(ctx, request.GetProviderId(), request.GetProviderIds())
	if err != nil {
		return nil, grpcError(err)
	}
	if len(providerIDs) == 0 {
		return &providerservicev1.ProbeProviderModelCatalogResponse{Message: "no providers to probe"}, nil
	}
	if err := s.runProviderCatalogDiscovery(ctx, providerIDs); err != nil {
		return nil, grpcError(err)
	}
	response := &providerservicev1.ProbeProviderModelCatalogResponse{
		ProviderIds: providerIDs,
		Message:     "provider model catalog probe completed",
	}
	if len(providerIDs) == 1 {
		response.ProviderId = providerIDs[0]
	}
	return response, nil
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

func (s *Server) providerProbeIDs(ctx context.Context, providerID string, providerIDs []string) ([]string, error) {
	ids := normalizedProviderIDs(providerIDs)
	if len(ids) == 0 && strings.TrimSpace(providerID) != "" {
		ids = []string{strings.TrimSpace(providerID)}
	}
	if len(ids) > 0 {
		return ids, nil
	}
	return s.providerIDs(ctx)
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
