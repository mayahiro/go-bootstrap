package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/fakegreeter"
)

func runBootstrap(ctx context.Context) error {
	config2, err := config.Load()
	if err != nil {
		return err
	}
	fakegreeterService := fakegreeter.New(config2)
	runParams := runParams{
		Greeter: fakegreeterService,
	}
	return run(ctx, runParams)
}
