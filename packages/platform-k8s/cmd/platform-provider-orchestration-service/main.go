package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authv1 "code-code.internal/go-contract/platform/auth/v1"
	oauthv1 "code-code.internal/go-contract/platform/oauth/v1"
	orchestrationv1 "code-code.internal/go-contract/platform/orchestration/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	platformk8s "code-code.internal/platform-k8s"
	"code-code.internal/platform-k8s/internal/platform/state"
	"code-code.internal/platform-k8s/internal/platform/telemetry"
	"code-code.internal/platform-k8s/internal/platform/temporalruntime"
	"code-code.internal/platform-k8s/internal/providerorchestration"
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
	grpcAddr := envOrDefault("PLATFORM_PROVIDER_ORCHESTRATION_SERVICE_GRPC_ADDR", ":8081")
	httpAddr := envOrDefault("PLATFORM_PROVIDER_ORCHESTRATION_SERVICE_HTTP_ADDR", ":8080")
	namespace := envOrDefault("PLATFORM_PROVIDER_ORCHESTRATION_SERVICE_NAMESPACE", "code-code")
	authAddr := envOrDefault("PLATFORM_PROVIDER_ORCHESTRATION_SERVICE_AUTH_GRPC_ADDR", "platform-auth-service:8081")
	providerAddr := envOrDefault("PLATFORM_PROVIDER_ORCHESTRATION_SERVICE_PROVIDER_GRPC_ADDR", "platform-provider-service:8081")
	databaseURL := firstEnv("PLATFORM_DATABASE_URL", "PLATFORM_PROVIDER_ORCHESTRATION_SERVICE_DATABASE_URL")

	telemetryShutdown, err := telemetry.Setup(context.Background(), envOrDefault("OTEL_SERVICE_NAME", "platform-provider-orchestration-service"))
	must(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryShutdown(ctx); err != nil {
			slog.Error("shutdown telemetry failed", "error", err)
		}
	}()

	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	must(err)
	defer func() { _ = authConn.Close() }()
	providerConn, err := grpc.NewClient(providerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	must(err)
	defer func() { _ = providerConn.Close() }()

	scheme := runtime.NewScheme()
	must(corev1.AddToScheme(scheme))
	must(platformk8s.AddToScheme(scheme))
	kubeConfig := ctrl.GetConfigOrDie()
	kubeClient, err := ctrlclient.New(kubeConfig, ctrlclient.Options{Scheme: scheme})
	must(err)
	statePool, err := state.OpenPostgres(context.Background(), databaseURL, "platform-provider-orchestration-service")
	must(err)
	defer statePool.Close()

	temporalConfig := temporalruntime.ConfigFromEnv(providerorchestration.TemporalTaskQueue)
	temporalClient, err := temporalruntime.Dial(context.Background(), temporalConfig)
	must(err)
	defer temporalClient.Close()
	providerConnect, err := providerorchestration.NewProviderConnectRuntime(providerorchestration.ConnectRuntimeConfig{
		Client:    kubeClient,
		Reader:    kubeClient,
		Namespace: namespace,
		StatePool: statePool,
		Auth:      authv1.NewAuthServiceClient(authConn),
		OAuth:     oauthv1.NewOAuthSessionServiceClient(authConn),
		Logger:    slog.Default(),
	})
	must(err)

	server, err := providerorchestration.NewServer(providerorchestration.Config{
		TemporalClient: temporalClient,
		TaskQueue:      temporalConfig.TaskQueue,
		Auth:           authv1.NewAuthServiceClient(authConn),
		Provider:       providerservicev1.NewProviderServiceClient(providerConn),
		Connect:        providerConnect,
		Logger:         slog.Default(),
	})
	must(err)
	temporalWorker := temporalruntime.NewWorker(temporalClient, temporalConfig.TaskQueue)
	must(providerorchestration.RegisterTemporalWorkflows(temporalWorker, server))
	must(providerorchestration.EnsureTemporalSchedules(context.Background(), temporalClient, temporalConfig.TaskQueue))
	must(temporalWorker.Start())
	defer temporalWorker.Stop()

	grpcListener, err := net.Listen("tcp", grpcAddr)
	must(err)
	httpListener, err := net.Listen("tcp", httpAddr)
	must(err)
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	orchestrationv1.RegisterProviderOrchestrationServiceServer(grpcServer, server)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(orchestrationv1.ProviderOrchestrationService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)

	httpServer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		http.NotFound(w, r)
	})}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	serveErr := make(chan error, 2)
	go func() { serveErr <- grpcServer.Serve(grpcListener) }()
	go func() {
		if err := httpServer.Serve(httpListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()
	go func() {
		<-ctx.Done()
		healthServer.SetServingStatus("", healthv1.HealthCheckResponse_NOT_SERVING)
		healthServer.SetServingStatus(orchestrationv1.ProviderOrchestrationService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_NOT_SERVING)
		grpcServer.GracefulStop()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	slog.Info("platform-provider-orchestration-service starting", "namespace", namespace, "grpc", grpcAddr, "http", httpAddr, "auth", authAddr, "provider", providerAddr, "temporal", temporalConfig.Address+"/"+temporalConfig.TaskQueue)
	select {
	case err := <-serveErr:
		must(err)
	case <-ctx.Done():
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
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

func must(err error) {
	if err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}
