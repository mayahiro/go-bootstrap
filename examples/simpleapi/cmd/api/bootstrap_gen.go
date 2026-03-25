package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/httpserver"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/logger"
)

func runBootstrap(ctx context.Context) error {
	config, err := config.Load()
	if err != nil {
		return err
	}
	logger := logger.New(config)
	server := httpserver.New(config, logger)
	if err := server.Start(ctx); err != nil {
		return err
	}
	defer func() { _ = server.Stop(ctx) }()
	return run(ctx, server)
}
