package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"syscall"

	"os"

	"golang.org/x/sync/errgroup"

	"terraforming-mars-manager/internal/framework/httpx"
	"terraforming-mars-manager/internal/framework/logx"
	"terraforming-mars-manager/internal/framework/signalx"
)

func main() {
	ctx := context.Background()
	cfg := MustNewConfig()

	// logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = logx.WithLogger(ctx, logger)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		router := http.NewServeMux()

		// Default route
		router.Handle("/", httputil.NewSingleHostReverseProxy(&cfg.GameURL.URL))

		logger.Info("starting http server", slog.String("addr", cfg.Listen))
		return httpx.ServeContext(ctx, router, cfg.Listen)
	})
	eg.Go(func() error {
		return signalx.ListenContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	})

	if err := eg.Wait(); err != nil {
		if errors.Is(err, signalx.ErrSignal) {
			logger.Info("signal received", slog.String("signal", err.Error()))
		} else {
			logger.Error("terminated with error", slog.String("error", err.Error()))
		}
	}
}
