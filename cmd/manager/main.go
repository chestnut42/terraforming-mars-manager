package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/sync/errgroup"

	"github.com/chestnut42/terraforming-mars-manager/internal/app"
	"github.com/chestnut42/terraforming-mars-manager/internal/database"
	"github.com/chestnut42/terraforming-mars-manager/internal/docs"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/signalx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

func main() {
	ctx := context.Background()
	cfg := MustNewConfig()

	// logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = logx.WithLogger(ctx, logger)

	db, err := database.PrepareDB(cfg.PostgresDSN)
	checkError(err)
	storageSvc, err := storage.New(db)
	checkError(err)

	docsSvc, err := docs.NewService()
	checkError(err)
	appSvc := app.NewService()

	grpcMux := runtime.NewServeMux()
	err = api.RegisterUsersHandlerServer(ctx, grpcMux, appSvc)
	checkError(err)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// App Router
		appRouter := http.NewServeMux()
		appRouter.Handle("/manager/api/", grpcMux)
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
