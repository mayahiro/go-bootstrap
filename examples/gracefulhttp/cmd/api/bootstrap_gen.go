package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/app"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/httpserver"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/logger"
)

func runBootstrap(ctx context.Context) error {
	config2, err := config.Load()
	if err != nil {
		return err
	}
	slogLogger := logger.New(config2)
	httpserverServer := httpserver.New(config2, slogLogger)
	appSignals := app.NewSignals()
	appRunParams := app.RunParams{
		Runner:  httpserverServer,
		Signals: appSignals,
	}
	if err := httpserverServer.Start(ctx); err != nil {
		return err
	}
	defer func() { _ = httpserverServer.Stop(ctx) }()
	if err := app.WatchSignals(ctx, appSignals); err != nil {
		return err
	}
	return app.Run(ctx, appRunParams)
}
