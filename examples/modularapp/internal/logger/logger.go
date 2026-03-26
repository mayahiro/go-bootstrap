package logger

import (
	"log/slog"
	"os"

	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/config"
)

func New(config *config.Config) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, nil)
	return slog.New(handler).With("app", config.Name)
}
