package audit

import (
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

func (writer *Writer) Close() error {
	fmt.Println("audit closed:", writer.name)
	return nil
}
