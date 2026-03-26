package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/prodgreeter"
)

func runBootstrap(ctx context.Context) error {
	config2, err := config.Load()
	if err != nil {
		return err
	}
	prodgreeterService := prodgreeter.New(config2)
	runParams := runParams{
		Greeter: prodgreeterService,
	}
	return run(ctx, runParams)
}
