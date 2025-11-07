package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"golang-grpc/internal/config"
	"golang-grpc/internal/service"
	grpctransport "golang-grpc/internal/transport/grpc"
	httptransport "golang-grpc/internal/transport/http"
	"golang-grpc/internal/user"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	store := user.NewStore()
	userService := service.NewUserService(store)

	grpcServer := grpctransport.NewServer(userService)
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}

	router := httptransport.NewRouter(userService)
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		log.Printf("gRPC server listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- err
		}
	}()

	go func() {
		log.Printf("HTTP server listening on http://%s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	graceCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGrace)
	defer cancel()

	stop()

	shutdown(graceCtx, grpcServer, httpServer)
}

func shutdown(ctx context.Context, grpcServer *grpc.Server, httpServer *http.Server) {
	done := make(chan struct{})

	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("http graceful shutdown failed: %v", err)
	}

	select {
	case <-done:
	case <-ctx.Done():
		log.Println("forcing gRPC shutdown")
		grpcServer.Stop()
	}

	// Ensuring any remaining requests finish before exit
	time.Sleep(250 * time.Millisecond)
	log.Println("servers shut down cleanly")
}
