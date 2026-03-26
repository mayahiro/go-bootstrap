//go:generate go tool bootstrapgen .

package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/audit"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/command"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/config"
)

type runParams struct {
	bootstrap.In
	Command *command.Command
}

var cliModule = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
		audit.NewWriter,
		command.New,
	),
	bootstrap.Lifecycle(
		bootstrap.Close((*audit.Writer)(nil)),
	),
)

var spec = bootstrap.CLI(
	"hello",
	bootstrap.Include(cliModule),
	bootstrap.Entry(run),
)

func run(ctx context.Context, params runParams) error {
	return params.Command.Run(ctx)
}
