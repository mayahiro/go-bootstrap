package main

import (
	"context"
	"log"
)

func main() {
	if err := runBootstrap(context.Background()); err != nil {
		log.Fatal(err)
	}
}
