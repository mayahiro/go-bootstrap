package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/httpserver"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/logger"
)

func runBootstrap(ctx context.Context) error {
	config2, err := config.Load()
	if err != nil {
		return err
	}
	slogLogger := logger.New(config2)
	httpserverServer := httpserver.New(config2, slogLogger)
	runParams := runParams{
		Runner: httpserverServer,
	}
	if err := httpserverServer.Start(ctx); err != nil {
		return err
	}
	defer func() { _ = httpserverServer.Stop(ctx) }()
	return run(ctx, runParams)
}
