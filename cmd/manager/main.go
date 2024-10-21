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
	"github.com/chestnut42/terraforming-mars-manager/internal/app/interceptor"
	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/database"
	"github.com/chestnut42/terraforming-mars-manager/internal/docs"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/signalx"
	"github.com/chestnut42/terraforming-mars-manager/internal/service/game"
	"github.com/chestnut42/terraforming-mars-manager/internal/service/notifications"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

func main() {
	ctx := context.Background()
	cfg := MustNewConfig()

	// logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = logx.WithLogger(ctx, logger)

	keyData, err := os.ReadFile(cfg.APN.KeyFile)
	checkError(err)

	db, err := database.PrepareDB(cfg.PostgresDSN)
	checkError(err)
	storageSvc, err := storage.New(db)
	checkError(err)
	httpClient := http.DefaultClient

	docsSvc, err := docs.NewService()
	checkError(err)
	marsSvc, err := mars.NewService(mars.Config{
		BaseURL:       cfg.GameURL.URL,
		PublicBaseURL: cfg.PublicGameURL.URL,
	}, httpClient)
	checkError(err)
	sandboxApnSvc, err := apn.NewService(apn.Config{
		BaseURL:     cfg.APN.SandboxURL.URL,
		Topic:       cfg.APN.BundleId,
		TeamId:      cfg.APN.TeamId,
		KeyId:       cfg.APN.KeyId,
		KeyData:     keyData,
		MaxTokenAge: cfg.APN.MaxTokenAge,
	}, httpClient)
	checkError(err)
	prodApnSvc, err := apn.NewService(apn.Config{
		BaseURL:     cfg.APN.ProdURL.URL,
		Topic:       cfg.APN.BundleId,
		TeamId:      cfg.APN.TeamId,
		KeyId:       cfg.APN.KeyId,
		KeyData:     keyData,
		MaxTokenAge: cfg.APN.MaxTokenAge,
	}, httpClient)
	checkError(err)
	originProxy := httputil.NewSingleHostReverseProxy(cfg.GameURL.URL)

	gameSvc := game.NewService(game.Config{
		ScanInterval: cfg.Games.ScanInterval,
	}, storageSvc, marsSvc)
	appSvc := app.NewService(storageSvc, gameSvc)
	authSvc, err := auth.NewService(ctx, cfg.AppleKeys)
	checkError(err)

	notifySvc := notifications.NewService(notifications.Config{
		ActivityBuffer: cfg.Notifications.ActivityBuffer,
		ScanInterval:   cfg.Notifications.ScanInterval,
		WorkersCount:   cfg.Notifications.WorkersCount,
	}, notifications.Dependencies{
		Storage:         storageSvc,
		Game:            gameSvc,
		SandboxNotifier: sandboxApnSvc,
		ProdNotifier:    prodApnSvc,
	})
	interceptorSvc := interceptor.NewService(originProxy, storageSvc, marsSvc, notifySvc)

	grpcMux := runtime.NewServeMux()
	err = api.RegisterUsersHandlerServer(ctx, grpcMux, appSvc)
	checkError(err)
	err = api.RegisterGamesHandlerServer(ctx, grpcMux, appSvc)
	checkError(err)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		apiHandler := httpx.WithAuthorization(grpcMux, authSvc)
		apiHandler = httpx.WithLogging(apiHandler)

		// App Router
		appRouter := http.NewServeMux()
		appRouter.Handle("/manager/api/", apiHandler)
		docsSvc.ConfigureRouter(appRouter, "/manager/docs")

		// Root Router
		root := http.NewServeMux()
		root.Handle("/manager/", appRouter)
		root.Handle("/", originProxy)

		// APIs to intercept
		root.HandleFunc("POST /player/input", interceptorSvc.PlayerInputHandler)

		logger.Info("starting http server", slog.String("addr", cfg.Listen))
		return httpx.ServeContext(ctx, httpx.WithRemoteAddress(root), cfg.Listen)
	})
	eg.Go(func() error {
		return notifySvc.Run(ctx)
	})
	eg.Go(func() error {
		return gameSvc.ProcessFinishedGames(ctx)
	})
	eg.Go(func() error {
		return gameSvc.ProcessElo(ctx)
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
