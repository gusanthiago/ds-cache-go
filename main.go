package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ds-cache/api"
	"ds-cache/config"
	"ds-cache/internal/cache"
	"ds-cache/internal/cache/lru"
	"ds-cache/internal/cache/tiered"
	"ds-cache/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("cache manager stopped", "error", err)
		os.Exit(1)
	}
}

// run wires the layers together and serves until interrupted.
// Returns an error instead of exiting, so every deferred cleanup still runs.
func run() error {
	// A missing .env is not fatal, the process may be configured through real
	// environment variables.
	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file loaded", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel})))
	if cfg.LogLevel > slog.LevelDebug {
		gin.SetMode(gin.ReleaseMode)
	}

	cacheService, err := service.NewCacheService(buildNodes(cfg))
	if err != nil {
		return fmt.Errorf("build cache service: %w", err)
	}

	server := &http.Server{
		Addr:           cfg.Port,
		Handler:        api.NewRouter(api.NewHandler(cacheService)),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return serve(server)
}

// buildNodes creates one node per configured slot, each backed by a small L1
// in front of a full size L2.
func buildNodes(cfg *config.Config) []cache.Store {
	nodes := make([]cache.Store, 0, cfg.Nodes)

	for i := 1; i <= cfg.Nodes; i++ {
		name := "node-" + strconv.Itoa(i)

		nodes = append(nodes, tiered.New(name,
			lru.New(name+"-l1", cfg.L1Capacity),
			lru.New(name+"-l2", cfg.NodeCapacity),
		))

		slog.Info("initialised cache node",
			"node", name,
			"l1_capacity", cfg.L1Capacity,
			"l2_capacity", cfg.NodeCapacity,
		)
	}

	return nodes
}

// serve runs the given server until SIGINT or SIGTERM, then drains it.
func serve(server *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("cache manager listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining connections")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
