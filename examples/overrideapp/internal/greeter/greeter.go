package greeter

import "context"

type Greeter interface {
	Greet(context.Context) error
}
