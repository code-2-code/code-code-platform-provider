package main

import (
	"context"
	"errors"
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
	"code-code.internal/platform-k8s/internal/providerservice"
	"connectrpc.com/connect"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	addr := envOrDefault("PLATFORM_PROVIDER_SERVICE_GRPC_ADDR", ":8081")
	httpAddr := envOrDefault("PLATFORM_PROVIDER_SERVICE_HTTP_ADDR", ":8080")
	namespace := envOrDefault("PLATFORM_PROVIDER_SERVICE_NAMESPACE", "code-code")
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
	must(coordinationv1.AddToScheme(scheme))
	must(platformk8s.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	kubeClient, err := ctrlclient.New(config, ctrlclient.Options{Scheme: scheme})
	must(err)
	statePool, err := state.OpenPostgres(context.Background(), databaseURL, "platform-provider-service")
	must(err)
	defer statePool.Close()
	server, err := providerservice.NewServer(providerservice.Config{
		Client:                          kubeClient,
		Reader:                          kubeClient,
		Namespace:                       namespace,
		StatePool:                       statePool,
		ProviderHostTelemetryMaxTargets: envIntOrDefault("PLATFORM_PROVIDER_SERVICE_HOST_TELEMETRY_MAX_TARGETS", 200),
	})
	must(err)
	listener, err := net.Listen("tcp", addr)
	must(err)
	httpListener, err := net.Listen("tcp", httpAddr)
	must(err)
	httpMux := http.NewServeMux()
	path, providerConnectHandler := providerservicev1connect.NewProviderServiceHandler(providerConnectHTTPAdapter{Server: server})
	httpMux.Handle(path, providerConnectHandler)
	httpMux.HandleFunc(providerservice.ProviderHostTelemetryTargetsPath, server.ServeProviderHostTelemetryTargets)
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

	slog.Info("platform-provider-service listening", "addr", addr, "namespace", namespace, "http", httpAddr)
	select {
	case err := <-serveErr:
		must(err)
	case <-ctx.Done():
		slog.Info("shutting down platform-provider-service")
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
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}
