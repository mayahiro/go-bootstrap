package health

import (
	"log/slog"

	"github.com/mayahiro/go-bootstrap/examples/modularapp/internal/config"
)

type Reporter struct {
	logger *slog.Logger
	addr   string
}

func NewReporter(config *config.Config, logger *slog.Logger) *Reporter {
	return &Reporter{
		logger: logger,
		addr:   config.Addr,
	}
}

func (reporter *Reporter) Report(status string) {
	reporter.logger.Info("health report", "status", status, "addr", reporter.addr)
}
