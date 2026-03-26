package command

import (
	"context"
	"fmt"

	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/audit"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/config"
)

type Command struct {
	name  string
	audit *audit.Writer
}

func New(config *config.Config, audit *audit.Writer) *Command {
	return &Command{
		name:  config.Name,
		audit: audit,
	}
}

func (command *Command) Run(context.Context) error {
	fmt.Println("hello from cli:", command.name, command.audit != nil)
	return nil
}
