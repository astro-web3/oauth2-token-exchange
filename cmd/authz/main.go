package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	httptransport "github.com/astro-web3/oauth2-token-exchange/internal/transport/http"
	"github.com/astro-web3/oauth2-token-exchange/pkg/otel"
)

const shutdownTimeoutSeconds = 10

func main() {
	cfg := config.MustLoad()

	srv, err := httptransport.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	serverErrChan := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on %s (mode: %s)", cfg.Server.Addr, cfg.Server.Mode)
		if listenErr := srv.ListenAndServe(); listenErr != nil &&
			!errors.Is(listenErr, http.ErrServerClosed) {
			log.Printf("Server failed: %v", listenErr)
			serverErrChan <- listenErr
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Shutting down server...")
	case serverErr := <-serverErrChan:
		log.Printf("Server error, shutting down: %v", serverErr)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		shutdownTimeoutSeconds*time.Second,
	)
	defer shutdownCancel()

	if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
		log.Printf("Server forced to shutdown: %v", shutdownErr)
	} else {
		log.Println("Server stopped gracefully")
	}

	if shutdownErr := otel.Shutdown(shutdownCtx); shutdownErr != nil {
		log.Printf("Failed to shutdown tracer provider: %v", shutdownErr)
	} else {
		log.Println("Tracer provider stopped gracefully")
	}
}
