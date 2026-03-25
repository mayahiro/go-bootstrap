package resolve

import (
	"bytes"
	"fmt"
	"go/types"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
)

type SourceKind string

const (
	ContextSource  SourceKind = "context"
	ProviderSource SourceKind = "provider"
)

type Source struct {
	Kind     SourceKind
	Type     types.Type
	Provider *model.Provider
}

type Step struct {
	Provider *model.Provider
	Inputs   []Source
}

type Lifecycle struct {
	Spec   model.Lifecycle
	Source Source
}

type Plan struct {
	Spec       *model.Spec
	Steps      []Step
	Entry      []Source
	Lifecycles []Lifecycle
}

func Build(spec *model.Spec) (*Plan, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec is required")
	}

	plan := &Plan{
		Spec: spec,
	}

	bindings := map[string]model.Binding{}
	for _, binding := range spec.Bindings {
		key := typeKey(binding.Interface)
		if existing, exists := bindings[key]; exists {
			return nil, duplicateBindingError(key, existing, binding)
		}

		bindings[key] = binding
	}

	seen := map[string]bool{}
	inputs := map[string][]Source{}
	state := map[string]int{}

	var resolveType func(types.Type, []requirement) (Source, error)
	resolveType = func(target types.Type, chain []requirement) (Source, error) {
		if isContextType(target) {
			return Source{
				Kind: ContextSource,
				Type: target,
			}, nil
		}

		provider, err := selectProvider(spec, bindings, target, chain)
		if err != nil {
			return Source{}, err
		}

		key := typeKey(provider.Output)

		switch state[key] {
		case 1:
			return Source{}, diagnosticError{
				message: fmt.Sprintf("cyclic dependency detected for %s", key),
				chain:   chain,
			}
		case 2:
			return Source{
				Kind:     ProviderSource,
				Type:     target,
				Provider: provider,
			}, nil
		}

		state[key] = 1

		dependencies := make([]Source, 0, len(provider.Inputs))
		for _, input := range provider.Inputs {
			source, err := resolveType(input, append(chain, providerRequirement(provider, input)))
			if err != nil {
				return Source{}, err
			}

			dependencies = append(dependencies, source)
		}

		state[key] = 2
		inputs[key] = dependencies

		if !seen[key] {
			plan.Steps = append(plan.Steps, Step{
				Provider: provider,
				Inputs:   dependencies,
			})
			seen[key] = true
		}

		return Source{
			Kind:     ProviderSource,
			Type:     target,
			Provider: provider,
		}, nil
	}

	for _, input := range spec.Entry.Inputs {
		source, err := resolveType(input, []requirement{entryRequirement(spec.Entry, input)})
		if err != nil {
			return nil, err
		}

		plan.Entry = append(plan.Entry, source)
	}

	for _, lifecycle := range spec.Lifecycles {
		source, err := resolveType(lifecycle.Target, []requirement{lifecycleRequirement(lifecycle)})
		if err != nil {
			return nil, err
		}

		plan.Lifecycles = append(plan.Lifecycles, Lifecycle{
			Spec:   lifecycle,
			Source: source,
		})
	}

	for index, step := range plan.Steps {
		plan.Steps[index].Inputs = inputs[typeKey(step.Provider.Output)]
	}

	return plan, nil
}

func selectProvider(spec *model.Spec, bindings map[string]model.Binding, target types.Type, chain []requirement) (*model.Provider, error) {
	mapped := target
	if binding, ok := bindings[typeKey(target)]; ok {
		mapped = binding.Implementation
	}

	exact := findProviders(spec.Providers, func(provider *model.Provider) bool {
		return typeKey(provider.Output) == typeKey(mapped)
	})

	switch len(exact) {
	case 1:
		return exact[0], nil
	case 0:
	default:
		return nil, providerSelectionError(fmt.Sprintf("multiple providers for %s", typeKey(mapped)), chain, exact)
	}

	assignable := findProviders(spec.Providers, func(provider *model.Provider) bool {
		return types.AssignableTo(provider.Output, target)
	})

	switch len(assignable) {
	case 1:
		return assignable[0], nil
	case 0:
		return nil, missingProviderError(target, chain, bindingNote(bindings, target))
	default:
		return nil, providerSelectionError(fmt.Sprintf("multiple assignable providers for %s", typeKey(target)), chain, assignable)
	}
}

func findProviders(providers []model.Provider, keep func(*model.Provider) bool) []*model.Provider {
	filtered := make([]*model.Provider, 0, len(providers))

	for index := range providers {
		provider := &providers[index]
		if keep(provider) {
			filtered = append(filtered, provider)
		}
	}

	return filtered
}

func isContextType(typ types.Type) bool {
	return typeKey(typ) == "context.Context"
}

func typeKey(typ types.Type) string {
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}

		return pkg.Path()
	})
}

type requirement struct {
	ownerKind string
	ownerName string
	position  model.Position
	target    types.Type
}

func (requirement requirement) String() string {
	return fmt.Sprintf("%s %s at %s requires %s", requirement.ownerKind, requirement.ownerName, requirement.position.String(), typeKey(requirement.target))
}

type diagnosticError struct {
	message    string
	chain      []requirement
	candidates []*model.Provider
	details    []string
}

func (err diagnosticError) Error() string {
	var body bytes.Buffer
	body.WriteString(err.message)

	if len(err.chain) > 0 {
		body.WriteString("\ndependency path:")
		for _, step := range err.chain {
			body.WriteString("\n- ")
			body.WriteString(step.String())
		}
	}

	if len(err.candidates) > 0 {
		body.WriteString("\ncandidates:")
		for _, candidate := range err.candidates {
			body.WriteString("\n- ")
			body.WriteString(candidate.Name)
			body.WriteString(" at ")
			body.WriteString(candidate.Position.String())
			body.WriteString(" returns ")
			body.WriteString(typeKey(candidate.Output))
		}
	}

	for _, detail := range err.details {
		body.WriteString("\n")
		body.WriteString(detail)
	}

	return body.String()
}

func entryRequirement(entry model.Entry, target types.Type) requirement {
	return requirement{
		ownerKind: "entry",
		ownerName: entry.Name,
		position:  entry.Position,
		target:    target,
	}
}

func providerRequirement(provider *model.Provider, target types.Type) requirement {
	return requirement{
		ownerKind: "provider",
		ownerName: provider.Name,
		position:  provider.Position,
		target:    target,
	}
}

func lifecycleRequirement(lifecycle model.Lifecycle) requirement {
	return requirement{
		ownerKind: "lifecycle",
		ownerName: lifecycleName(lifecycle),
		position:  lifecycle.Position,
		target:    lifecycle.Target,
	}
}

func lifecycleName(lifecycle model.Lifecycle) string {
	switch lifecycle.Kind {
	case model.StartStopLifecycle:
		return "StartStop"
	case model.CloseLifecycle:
		return "Close"
	default:
		return string(lifecycle.Kind)
	}
}

func duplicateBindingError(key string, existing model.Binding, current model.Binding) error {
	return diagnosticError{
		message: fmt.Sprintf("duplicate binding for %s", key),
		details: []string{
			fmt.Sprintf("first binding at %s", existing.Position.String()),
			fmt.Sprintf("second binding at %s", current.Position.String()),
		},
	}
}

func providerSelectionError(message string, chain []requirement, candidates []*model.Provider) error {
	return diagnosticError{
		message:    message,
		chain:      chain,
		candidates: candidates,
	}
}

func missingProviderError(target types.Type, chain []requirement, note string) error {
	details := []string{}
	if note != "" {
		details = append(details, note)
	}

	return diagnosticError{
		message: fmt.Sprintf("provider not found for %s", typeKey(target)),
		chain:   chain,
		details: details,
	}
}

func bindingNote(bindings map[string]model.Binding, target types.Type) string {
	binding, ok := bindings[typeKey(target)]
	if !ok {
		return ""
	}

	return fmt.Sprintf("binding at %s maps %s to %s", binding.Position.String(), typeKey(binding.Interface), typeKey(binding.Implementation))
}
