package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/chestnut42/terraforming-mars-manager/internal/docs"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/signalx"
)

func main() {
	ctx := context.Background()
	cfg := MustNewConfig()

	// logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = logx.WithLogger(ctx, logger)

	docsSvc, err := docs.NewService()
	checkError(err)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// App Router
		appRouter := http.NewServeMux()
		docsSvc.ConfigureRouter(appRouter, "/manager/docs")

		// Root Router
		root := http.NewServeMux()
		root.Handle("/manager/", appRouter)
		root.Handle("/", httputil.NewSingleHostReverseProxy(&cfg.GameURL.URL))

		logger.Info("starting http server", slog.String("addr", cfg.Listen))
		return httpx.ServeContext(ctx, root, cfg.Listen)
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

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
