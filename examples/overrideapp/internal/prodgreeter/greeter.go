package prodgreeter

import (
	"context"
	"fmt"

	"github.com/mayahiro/go-bootstrap/examples/overrideapp/internal/config"
)

type Service struct {
	name string
}

func New(config *config.Config) *Service {
	return &Service{
		name: config.Name,
	}
}

func (service *Service) Greet(context.Context) error {
	fmt.Println("hello from prod:", service.name)
	return nil
}
