package app

import (
	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/config"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/fakegreeter"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/greeter"
	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/prodgreeter"
)

var BaseModule = bootstrap.Module(
	bootstrap.Provide(
		config.Load,
		prodgreeter.New,
	),
	bootstrap.Bind(
		(*greeter.Greeter)(nil),
		(*prodgreeter.Service)(nil),
	),
)

var TestModule = bootstrap.Module(
	bootstrap.Include(BaseModule),
	bootstrap.Override(
		bootstrap.Provide(
			fakegreeter.New,
		),
		bootstrap.Bind(
			(*greeter.Greeter)(nil),
			(*fakegreeter.Service)(nil),
		),
	),
)
