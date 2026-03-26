package main

import (
	"context"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/audit"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/command"
	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/config"
)

func runBootstrap(ctx context.Context) error {
	config2, err := config.Load()
	if err != nil {
		return err
	}
	auditWriter := audit.NewWriter(config2)
	command2 := command.New(config2, auditWriter)
	runParams := runParams{
		Command: command2,
	}
	defer func() { _ = audit.Flush(ctx, auditWriter) }()
	return run(ctx, runParams)
}
