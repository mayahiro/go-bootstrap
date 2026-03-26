package audit

import (
	"context"
	"fmt"

	"github.com/mayahiro/go-bootstrap/examples/simplecli/internal/config"
)

type Writer struct {
	name string
}

func NewWriter(config *config.Config) *Writer {
	return &Writer{
		name: config.Name,
	}
}

func Flush(context.Context, *Writer) error {
	fmt.Println("audit flushed")
	return nil
}
