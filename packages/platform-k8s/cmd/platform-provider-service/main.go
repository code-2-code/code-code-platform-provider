package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/go-contract/platform/provider/v1/providerservicev1connect"
	platformk8s "code-code.internal/platform-k8s"
	"code-code.internal/platform-k8s/internal/platform/state"
	"code-code.internal/platform-k8s/internal/platform/telemetry"
	"code-code.internal/platform-k8s/internal/platform/temporalruntime"
	"code-code.internal/platform-k8s/internal/platform/triggerhttp"
	"code-code.internal/platform-k8s/internal/providerservice"
	"code-code.internal/platform-k8s/internal/providerservice/providerconnect"
	"connectrpc.com/connect"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	addr := envOrDefault("PLATFORM_PROVIDER_SERVICE_GRPC_ADDR", ":8081")
	httpAddr := envOrDefault("PLATFORM_PROVIDER_SERVICE_HTTP_ADDR", ":8080")
	namespace := envOrDefault("PLATFORM_PROVIDER_SERVICE_NAMESPACE", "code-code")
	authAddr := envOrDefault("PLATFORM_PROVIDER_SERVICE_AUTH_GRPC_ADDR", "platform-auth-service:8081")
	modelAddr := envOrDefault("PLATFORM_PROVIDER_SERVICE_MODEL_GRPC_ADDR", "platform-model-service:8081")
	internalActionToken := strings.TrimSpace(os.Getenv("PLATFORM_PROVIDER_SERVICE_INTERNAL_ACTION_TOKEN"))
	databaseURL := firstEnv("PLATFORM_DATABASE_URL", "PLATFORM_PROVIDER_SERVICE_DATABASE_URL")

	telemetryShutdown, err := telemetry.Setup(context.Background(), envOrDefault("OTEL_SERVICE_NAME", "platform-provider-service"))
	must(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = telemetryShutdown(ctx)
	}()

	scheme := runtime.NewScheme()
	must(corev1.AddToScheme(scheme))
	must(platformk8s.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	kubeClient, err := ctrlclient.New(config, ctrlclient.Options{Scheme: scheme})
	must(err)
	statePool, err := state.OpenPostgres(context.Background(), databaseURL, "platform-provider-service")
	must(err)
	defer statePool.Close()

	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	must(err)
	defer func() { _ = authConn.Close() }()
	modelConn, err := grpc.NewClient(modelAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	must(err)
	defer func() { _ = modelConn.Close() }()
	temporalConfig := temporalruntime.ConfigFromEnv(providerconnect.TemporalTaskQueue)
	temporalClient, err := temporalruntime.Dial(context.Background(), temporalConfig)
	must(err)
	defer temporalClient.Close()
	postConnectRuntime, err := providerconnect.NewTemporalPostConnectWorkflowRuntime(providerconnect.TemporalPostConnectWorkflowRuntimeConfig{
		Client:                  temporalClient,
		TaskQueue:               temporalConfig.TaskQueue,
		PlatformNamespace:       namespace,
		ProviderHTTPBaseURL:     envOrDefault("PLATFORM_PROVIDER_SERVICE_WORKFLOW_PROVIDER_HTTP_BASE_URL", "http://platform-provider-service.code-code.svc.cluster.local:8080/internal/actions"),
		ProviderHTTPActionToken: internalActionToken,
	})
	must(err)

	server, err := providerservice.NewServer(providerservice.Config{
		Client:                             kubeClient,
		Reader:                             kubeClient,
		Namespace:                          namespace,
		AuthConn:                           authConn,
		ModelConn:                          modelConn,
		StatePool:                          statePool,
		ProviderConnectProviderHTTPBaseURL: postConnectRuntime.ProviderHTTPBaseURL(),
		ProviderHostTelemetryMaxTargets:    envIntOrDefault("PLATFORM_PROVIDER_SERVICE_HOST_TELEMETRY_MAX_TARGETS", 200),
		PostConnect:                        postConnectRuntime,
	})
	must(err)
	temporalWorker := temporalruntime.NewWorker(temporalClient, temporalConfig.TaskQueue)
	must(postConnectRuntime.Register(temporalWorker))
	must(providerservice.RegisterTemporalWorkflows(temporalWorker, server))
	must(providerservice.EnsureTemporalSchedules(context.Background(), temporalClient, temporalConfig.TaskQueue))
	must(temporalWorker.Start())
	defer temporalWorker.Stop()
	triggerServer, err := triggerhttp.NewServer(triggerhttp.Config{
		Logger: slog.Default(),
		Actions: map[string]triggerhttp.ActionFunc{
			"discover-provider-catalogs": func(ctx context.Context, request triggerhttp.Request) (any, error) {
				body, err := decodeProviderTriggerBody(request)
				if err != nil {
					return nil, err
				}
				if err := server.DiscoverProviderCatalogs(ctx, body.ProviderIDs); err != nil {
					return nil, err
				}
				return map[string]any{"provider_ids": body.ProviderIDs}, nil
			},
			"bind-provider-catalogs": func(ctx context.Context, _ triggerhttp.Request) (any, error) {
				response, err := server.BindProviderCatalogs(ctx, &providerservicev1.BindProviderCatalogsRequest{})
				if err != nil {
					return nil, err
				}
				return map[string]string{"status": response.GetStatus()}, nil
			},
			"submit-provider-observability-probe": func(ctx context.Context, request triggerhttp.Request) (any, error) {
				body, err := decodeProviderTriggerBody(request)
				if err != nil {
					return nil, err
				}
				response, err := server.ProbeProviderObservability(ctx, &providerservicev1.ProbeProviderObservabilityRequest{
					ProviderIds: body.ProviderIDs,
					Trigger:     providerTrigger(body.Trigger),
				})
				if err != nil {
					return nil, err
				}
				return map[string]any{"provider_ids": response.GetProviderIds(), "message": response.GetMessage()}, nil
			},
		},
		AuthToken: internalActionToken,
	})
	must(err)
	if internalActionToken == "" {
		log.Printf("internal action endpoints are disabled (env PLATFORM_PROVIDER_SERVICE_INTERNAL_ACTION_TOKEN is empty)")
	}

	listener, err := net.Listen("tcp", addr)
	must(err)
	httpListener, err := net.Listen("tcp", httpAddr)
	must(err)
	httpMux := http.NewServeMux()
	_, providerConnectHandler := providerservicev1connect.NewProviderServiceHandler(providerConnectHTTPAdapter{Server: server})
	httpMux.Handle(providerservicev1connect.ProviderServiceListVendorsProcedure, providerConnectHandler)
	httpMux.HandleFunc(providerservice.ProviderHostTelemetryTargetsPath, server.ServeProviderHostTelemetryTargets)
	httpMux.Handle("/", triggerServer)
	httpServer := &http.Server{Handler: httpMux}
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	providerservicev1.RegisterProviderServiceServer(grpcServer, server)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(providerservicev1.ProviderService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	serveErr := make(chan error, 2)
	go func() {
		if err := httpServer.Serve(httpListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()
	go func() { serveErr <- grpcServer.Serve(listener) }()

	log.Printf("platform-provider-service listening on %s (namespace=%s auth=%s model=%s http=%s)", addr, namespace, authAddr, modelAddr, httpAddr)
	select {
	case err := <-serveErr:
		must(err)
	case <-ctx.Done():
		log.Println("shutting down platform-provider-service...")
		healthServer.SetServingStatus("", healthv1.HealthCheckResponse_NOT_SERVING)
		healthServer.SetServingStatus(providerservicev1.ProviderService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_NOT_SERVING)
		grpcServer.Stop()
		_ = httpServer.Close()
	}
}

type providerConnectHTTPAdapter struct {
	*providerservice.Server
}

func (a providerConnectHTTPAdapter) WatchProviderStatusEvents(
	ctx context.Context,
	request *providerservicev1.WatchProviderStatusEventsRequest,
	stream *connect.ServerStream[providerservicev1.WatchProviderStatusEventsResponse],
) error {
	return a.Server.StreamProviderStatusEvents(ctx, request.GetProviderIds(), func(event *providerservicev1.ProviderStatusEvent) error {
		return stream.Send(&providerservicev1.WatchProviderStatusEventsResponse{Event: event})
	})
}

type providerTriggerBody struct {
	ProviderIDs []string `json:"provider_ids"`
	ProviderID  string   `json:"provider_id"`
	Trigger     string   `json:"trigger"`
}

func decodeProviderTriggerBody(request triggerhttp.Request) (providerTriggerBody, error) {
	var body providerTriggerBody
	if err := request.DecodeJSON(&body); err != nil {
		return providerTriggerBody{}, err
	}
	body.ProviderIDs = normalizeProviderIDs(append(body.ProviderIDs, body.ProviderID))
	return body, nil
}

func normalizeProviderIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func providerTrigger(trigger string) providerservicev1.ProviderObservabilityProbeTrigger {
	switch strings.TrimSpace(trigger) {
	case "connect":
		return providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_CONNECT
	case "schedule":
		return providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_SCHEDULE
	default:
		return providerservicev1.ProviderObservabilityProbeTrigger_PROVIDER_OBSERVABILITY_PROBE_TRIGGER_MANUAL
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
