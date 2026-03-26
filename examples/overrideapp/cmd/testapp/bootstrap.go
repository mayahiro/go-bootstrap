//go:generate go tool bootstrapgen .

package main

import (
	"context"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/app"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/greeter"
)

type runParams struct {
	bootstrap.In
	Greeter greeter.Greeter
}

var spec = bootstrap.CLI(
	"overrideapp-test",
	bootstrap.Include(app.TestModule),
	bootstrap.Entry(run),
)

func run(ctx context.Context, params runParams) error {
	return params.Greeter.Greet(ctx)
}
