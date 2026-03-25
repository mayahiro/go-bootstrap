//go:generate go tool bootstrapgen .

package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/httpserver"
	examplelogger "github.com/mayahiro/go-bootstrap/examples/simpleapi/internal/logger"
)

var spec = bootstrap.Server(
	"api",
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
	bootstrap.Entry(run),
)

func run(ctx context.Context, runner httpserver.Runner) error {
	return runner.Run(ctx)
}
