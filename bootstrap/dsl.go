package bootstrap

type AppKind string

const (
	ServerKind AppKind = "server"
	CLIKind    AppKind = "cli"
)

type options struct {
	Providers  []any
	Bindings   []Binding
	Entry      any
	Lifecycles []Hook
	Includes   []ModuleSpec
}

type Spec struct {
	Kind AppKind
	Name string
	options
}

type ModuleSpec struct {
	options
}

type Binding struct {
	Interface      any
	Implementation any
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(target *options) {
	f(target)
}

func Server(name string, options ...Option) Spec {
	return newSpec(ServerKind, name, options...)
}

func CLI(name string, options ...Option) Spec {
	return newSpec(CLIKind, name, options...)
}

func Provide(constructors ...any) Option {
	return optionFunc(func(target *options) {
		target.Providers = append(target.Providers, constructors...)
	})
}

func Bind(iface any, implementation any) Option {
	return optionFunc(func(target *options) {
		target.Bindings = append(target.Bindings, Binding{
			Interface:      iface,
			Implementation: implementation,
		})
	})
}

func Entry(entry any) Option {
	return optionFunc(func(target *options) {
		target.Entry = entry
	})
}

func Lifecycle(hooks ...Hook) Option {
	return optionFunc(func(target *options) {
		target.Lifecycles = append(target.Lifecycles, hooks...)
	})
}

func Module(options ...Option) ModuleSpec {
	module := ModuleSpec{}
	for _, option := range options {
		if option == nil {
			continue
		}

		option.apply(&module.options)
	}
	return module
}

func Include(modules ...ModuleSpec) Option {
	return optionFunc(func(target *options) {
		target.Includes = append(target.Includes, modules...)
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

		option.apply(&spec.options)
	}

	return spec
}
