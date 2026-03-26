//go:generate go tool bootstrapgen .

package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/httpserver"
	examplelogger "github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/logger"
)

type runParams struct {
	bootstrap.In
	Runner httpserver.Runner
}

var serverModule = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
		examplelogger.New,
		httpserver.New,
	),
	bootstrap.Bind(
		(*httpserver.Runner)(nil),
		(*httpserver.Server)(nil),
	),
	bootstrap.Lifecycle(
		bootstrap.StartStop((*httpserver.Server)(nil), "Start", "Stop"),
	),
)

var spec = bootstrap.Server(
	"api",
	bootstrap.Include(serverModule),
	bootstrap.Entry(run),
)

func run(ctx context.Context, params runParams) error {
	return params.Runner.Run(ctx)
}
