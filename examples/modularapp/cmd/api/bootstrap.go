//go:generate go tool bootstrapgen .

package main

import (
	"context"
	"log/slog"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/app"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/health"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/httpserver"
)

type serverParams struct {
	bootstrap.In
	Runner httpserver.Runner
}

type observabilityParams struct {
	bootstrap.In
	Logger   *slog.Logger
	Reporter *health.Reporter
}

var spec = bootstrap.Server(
	"modular-api",
	bootstrap.Include(
		app.ConfigModule,
		app.LoggingModule,
		app.HealthModule,
		app.ServerModule,
	),
	bootstrap.Entry(run),
)

func run(ctx context.Context, server serverParams, observability observabilityParams) error {
	observability.Logger.InfoContext(ctx, "starting modular app")
	observability.Reporter.Report("ready")
	return server.Runner.Run(ctx)
}
