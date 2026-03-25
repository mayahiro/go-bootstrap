package model

import (
	"fmt"
	"go/types"
)

type Position struct {
	File   string
	Line   int
	Column int
}

func (position Position) String() string {
	switch {
	case position.File == "" && position.Line == 0 && position.Column == 0:
		return "<unknown>"
	case position.Column == 0:
		return fmt.Sprintf("%s:%d", position.File, position.Line)
	default:
		return fmt.Sprintf("%s:%d:%d", position.File, position.Line, position.Column)
	}
}

type Spec struct {
	Kind        string
	Name        string
	Position    Position
	PackageName string
	PackagePath string
	Directory   string
	Providers   []Provider
	Bindings    []Binding
	Entry       Entry
	Lifecycles  []Lifecycle
}

type Provider struct {
	Name        string
	Position    Position
	PackageName string
	PackagePath string
	Inputs      []types.Type
	Output      types.Type
	HasError    bool
}

type Entry struct {
	Name         string
	Position     Position
	PackageName  string
	PackagePath  string
	Inputs       []types.Type
	ReturnsError bool
}

type Binding struct {
	Interface      types.Type
	Implementation types.Type
	Position       Position
}

type LifecycleKind string

const (
	CloseLifecycle     LifecycleKind = "close"
	StartStopLifecycle LifecycleKind = "start_stop"
)

type Lifecycle struct {
	Kind     LifecycleKind
	Target   types.Type
	Start    string
	Stop     string
	Position Position
}
