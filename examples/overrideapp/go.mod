module github.com/mayahiro/go-bootstrap/examples/overrideapp

go 1.26.1

require github.com/mayahiro/go-bootstrap v0.0.0

require (
	github.com/mayahiro/go-bootstrap/bootstrapgen v0.0.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
)

tool github.com/mayahiro/go-bootstrap/bootstrapgen/cmd/bootstrapgen

replace github.com/mayahiro/go-bootstrap => ../..

replace github.com/mayahiro/go-bootstrap/bootstrapgen => ../../bootstrapgen
