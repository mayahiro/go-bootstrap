package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mayahiro/go-bootstrap/bootstrap"
	"github.com/mayahiro/go-bootstrap/examples/gracefulhttp/internal/httpserver"
)

type RunParams struct {
	bootstrap.In
	Runner  httpserver.Runner
	Signals *Signals
}

type Signals struct {
	done chan struct{}
}

func NewSignals() *Signals {
	return &Signals{
		done: make(chan struct{}),
	}
}

func WatchSignals(ctx context.Context, signals *Signals) error {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		defer signal.Stop(ch)
		select {
		case <-ctx.Done():
		case <-ch:
			close(signals.done)
		}
	}()

	return nil
}

func Run(ctx context.Context, params RunParams) error {
	if err := params.Runner.Run(ctx); err != nil {
		return err
	}

	<-params.Signals.done
	return nil
}
