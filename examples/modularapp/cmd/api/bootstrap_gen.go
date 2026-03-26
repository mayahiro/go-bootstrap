package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/health"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/httpserver"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/logger"
)

func runBootstrap(ctx context.Context) error {
	config2, err := config.Load()
	if err != nil {
		return err
	}
	slogLogger := logger.New(config2)
	httpserverServer := httpserver.New(config2, slogLogger)
	healthReporter := health.NewReporter(config2, slogLogger)
	serverParams := serverParams{
		Runner: httpserverServer,
	}
	observabilityParams := observabilityParams{
		Logger:   slogLogger,
		Reporter: healthReporter,
	}
	if err := httpserverServer.Start(ctx); err != nil {
		return err
	}
	defer func() { _ = httpserverServer.Stop(ctx) }()
	return run(ctx, serverParams, observabilityParams)
}
