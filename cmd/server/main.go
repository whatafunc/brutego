// Package main provides the app's entry point, setting up the gRPC server and HTTP gateway.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/whatafunc/brutego/internal/config"
	"github.com/whatafunc/brutego/internal/service"
	"github.com/whatafunc/brutego/internal/storage"
	pb "github.com/whatafunc/brutego/pkg/api/antibruteforce/v1"
)

func main() {
	logger := mustInitLogger()
	defer func() { _ = logger.Sync() }()

	cfg := mustLoadConfig(logger)

	svc := buildService(logger, cfg)

	grpcServer, grpcLis := mustCreateGRPCServer(logger, cfg, svc)
	httpServer := mustCreateHTTPServer(logger, cfg)

	run(logger, grpcServer, grpcLis, httpServer)
}

// -----------------------------------------------------------------------------
// Bootstrap
// -----------------------------------------------------------------------------

func mustInitLogger() *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	return logger
}

func mustLoadConfig(logger *zap.Logger) *config.Config {
	cfg, err := config.New()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	return cfg
}

func buildService(logger *zap.Logger, cfg *config.Config) pb.AntiBruteforceServiceServer {
	store := storage.NewMemoryStorage()

	return service.New(service.Deps{
		Logger:  logger,
		Storage: store,
		Limits: service.Limits{
			LoginRPM:    cfg.LoginRPM,
			PasswordRPM: cfg.PasswordRPM,
			IPRPM:       cfg.IPRPM,
		},
	})
}

// -----------------------------------------------------------------------------
// Servers
// -----------------------------------------------------------------------------

func mustCreateGRPCServer(
	logger *zap.Logger,
	cfg *config.Config,
	svc pb.AntiBruteforceServiceServer,
) (*grpc.Server, net.Listener) {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor(logger)),
	)

	pb.RegisterAntiBruteforceServiceServer(server, svc)
	reflection.Register(server)

	ctx := context.Background()
	lc := &net.ListenConfig{}
	grpcLis, err := lc.Listen(ctx, "tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatal("failed to listen gRPC", zap.String("addr", cfg.GRPCAddr), zap.Error(err))
	}

	return server, grpcLis
}

func mustCreateHTTPServer(logger *zap.Logger, cfg *config.Config) *http.Server {
	ctx := context.Background()

	gwMux := runtime.NewServeMux()
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := pb.RegisterAntiBruteforceServiceHandlerFromEndpoint(
		ctx, gwMux, cfg.GRPCAddr, dialOpts,
	); err != nil {
		logger.Fatal("failed to register gateway", zap.Error(err))
	}

	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      gwMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// -----------------------------------------------------------------------------
// Run & Shutdown
// -----------------------------------------------------------------------------

func run(
	logger *zap.Logger,
	grpcServer *grpc.Server,
	grpcLis net.Listener,
	httpServer *http.Server,
) {
	errCh := make(chan error, 2)

	go func() {
		logger.Info("gRPC server listening", zap.String("addr", grpcLis.Addr().String()))
		if err := grpcServer.Serve(grpcLis); err != nil {
			errCh <- fmt.Errorf("gRPC server: %w", err)
		}
	}()

	go func() {
		logger.Info("HTTP gateway listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("HTTP gateway: %w", err)
		}
	}()

	waitForShutdown(logger, grpcServer, httpServer, errCh)
}

func waitForShutdown(
	logger *zap.Logger,
	grpcServer *grpc.Server,
	httpServer *http.Server,
	errCh chan error,
) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case sig := <-quit:
		logger.Info("shutting down", zap.String("signal", sig.String()))
	case err := <-errCh:
		logger.Error("server error", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP gateway shutdown error", zap.Error(err))
	}

	logger.Info("shutdown complete")
}

// -----------------------------------------------------------------------------
// Interceptors
// -----------------------------------------------------------------------------

func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		logger.Info("gRPC call",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		return resp, err
	}
}