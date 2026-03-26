//go:generate go tool bootstrapgen .

package main

import (
	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/app"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/httpserver"
	examplelogger "github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/logger"
)

var serverModule = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
		examplelogger.New,
		httpserver.New,
		app.NewSignals,
	),
	bootstrap.Bind(
		(*httpserver.Runner)(nil),
		(*httpserver.Server)(nil),
	),
	bootstrap.Lifecycle(
		bootstrap.StartStop((*httpserver.Server).Start, (*httpserver.Server).Stop),
		bootstrap.HookFunc(app.WatchSignals, nil),
	),
)

var spec = bootstrap.Server(
	"graceful-http",
	bootstrap.Include(serverModule),
	bootstrap.Entry(app.Run),
)
