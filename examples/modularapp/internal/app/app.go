package app

import (
	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/health"
	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/httpserver"
	examplelogger "github.com/mayahiro/go-bootstrap/examples/modularapp/internal/logger"
)

var ConfigModule = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
	),
)

var LoggingModule = bootstrap.Module(
	bootstrap.Provide(
		examplelogger.New,
	),
)

var HealthModule = bootstrap.Module(
	bootstrap.Provide(
		health.NewReporter,
	),
)

var ServerModule = bootstrap.Module(
	bootstrap.Provide(
		httpserver.New,
	),
	bootstrap.Bind(
		(*httpserver.Runner)(nil),
		(*httpserver.Server)(nil),
	),
	bootstrap.Lifecycle(
		bootstrap.StartStop((*httpserver.Server).Start, (*httpserver.Server).Stop),
	),
)
