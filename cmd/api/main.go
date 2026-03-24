package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kisotu/neobank/internal/container"
)

func main() {
	ctx := context.Background()

	appContainer, err := container.Build(ctx)
	if err != nil {
		log.Fatalf("failed to build container: %v", err)
	}
	defer appContainer.Close()

	addr := fmt.Sprintf("%s:%s", appContainer.Config.Server.Host, appContainer.Config.Server.Port)

	server := &http.Server{
		Addr:              addr,
		Handler:           appContainer.HTTPHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
