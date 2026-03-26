package httpserver

import (
	"context"
	"log/slog"

	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/config"
)

type Runner interface {
	Run(context.Context) error
}

type Server struct {
	addr   string
	logger *slog.Logger
}

func New(config *config.Config, logger *slog.Logger) *Server {
	return &Server{
		addr:   config.Addr,
		logger: logger,
	}
}

func (server *Server) Start(ctx context.Context) error {
	server.logger.InfoContext(ctx, "server started", "addr", server.addr)
	return nil
}

func (server *Server) Run(ctx context.Context) error {
	server.logger.InfoContext(ctx, "waiting for shutdown", "addr", server.addr)
	return nil
}

func (server *Server) Stop(ctx context.Context) error {
	server.logger.InfoContext(ctx, "server stopped", "addr", server.addr)
	return nil
}
