package bootstrap

type AppKind string

const (
	ServerKind AppKind = "server"
	CLIKind    AppKind = "cli"
)

type Spec struct {
	Kind       AppKind
	Name       string
	Providers  []any
	Bindings   []Binding
	Entry      any
	Lifecycles []Hook
}

type Binding struct {
	Interface      any
	Implementation any
}

type Option interface {
	apply(*Spec)
}

type optionFunc func(*Spec)

func (f optionFunc) apply(spec *Spec) {
	f(spec)
}

func Server(name string, options ...Option) Spec {
	return newSpec(ServerKind, name, options...)
}

func CLI(name string, options ...Option) Spec {
	return newSpec(CLIKind, name, options...)
}

func Provide(constructors ...any) Option {
	return optionFunc(func(spec *Spec) {
		spec.Providers = append(spec.Providers, constructors...)
	})
}

func Bind(iface any, implementation any) Option {
	return optionFunc(func(spec *Spec) {
		spec.Bindings = append(spec.Bindings, Binding{
			Interface:      iface,
			Implementation: implementation,
		})
	})
}

func Entry(entry any) Option {
	return optionFunc(func(spec *Spec) {
		spec.Entry = entry
	})
}

func Lifecycle(hooks ...Hook) Option {
	return optionFunc(func(spec *Spec) {
		spec.Lifecycles = append(spec.Lifecycles, hooks...)
	})
}

func newSpec(kind AppKind, name string, options ...Option) Spec {
	spec := Spec{
		Kind: kind,
		Name: name,
	}

	for _, option := range options {
		if option == nil {
			continue
		}

		option.apply(&spec)
	}

	return spec
}
