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

type EntryField struct {
	Name   string
	Source Source
}

type EntryArg struct {
	Type   types.Type
	Source Source
	Fields []EntryField
}

type HookCall struct {
	Func   *model.Function
	Inputs []Source
}

type Lifecycle struct {
	Spec   model.Lifecycle
	Source Source
	Start  *HookCall
	Stop   *HookCall
}

type Plan struct {
	Spec       *model.Spec
	Steps      []Step
	Entry      []EntryArg
	Lifecycles []Lifecycle
}

func Build(spec *model.Spec) (*Plan, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec is required")
	}

	plan := &Plan{
		Spec: spec,
	}

	bindings, err := bindingMap(spec.Bindings)
	if err != nil {
		return nil, err
	}

	overrideBindings, err := bindingMap(spec.OverrideBindings)
	if err != nil {
		return nil, err
	}

	overrideProviders, err := providerMap(spec.Overrides)
	if err != nil {
		return nil, err
	}

	normalProviders, err := providerMap(spec.Providers)
	if err != nil {
		return nil, err
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

		provider, err := selectProvider(spec, overrideBindings, bindings, overrideProviders, normalProviders, target, chain)
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
		if len(input.Fields) == 0 {
			source, err := resolveType(input.Type, []requirement{entryRequirement(spec.Entry, input.Type)})
			if err != nil {
				return nil, err
			}

			plan.Entry = append(plan.Entry, EntryArg{
				Type:   input.Type,
				Source: source,
			})
			continue
		}

		arg := EntryArg{
			Type: input.Type,
		}

		for _, field := range input.Fields {
			source, err := resolveType(field.Type, []requirement{entryFieldRequirement(spec.Entry, field)})
			if err != nil {
				return nil, err
			}

			arg.Fields = append(arg.Fields, EntryField{
				Name:   field.Name,
				Source: source,
			})
		}

		plan.Entry = append(plan.Entry, arg)
	}

	for _, lifecycle := range spec.Lifecycles {
		resolved := Lifecycle{
			Spec: lifecycle,
		}

		switch lifecycle.Kind {
		case model.CloseLifecycle, model.StartStopLifecycle:
			source, err := resolveType(lifecycle.Target, []requirement{lifecycleRequirement(lifecycle)})
			if err != nil {
				return nil, err
			}

			resolved.Source = source
		case model.HookFuncLifecycle:
			start, err := resolveHook(resolveType, lifecycle.OnStart)
			if err != nil {
				return nil, err
			}

			stop, err := resolveHook(resolveType, lifecycle.OnStop)
			if err != nil {
				return nil, err
			}

			resolved.Start = start
			resolved.Stop = stop
		}

		plan.Lifecycles = append(plan.Lifecycles, resolved)
	}

	for index, step := range plan.Steps {
		plan.Steps[index].Inputs = inputs[typeKey(step.Provider.Output)]
	}

	return plan, nil
}

func resolveHook(resolveType func(types.Type, []requirement) (Source, error), fn *model.Function) (*HookCall, error) {
	if fn == nil {
		return nil, nil
	}

	call := &HookCall{
		Func: fn,
	}

	for _, input := range fn.Inputs {
		source, err := resolveType(input, []requirement{hookRequirement(fn, input)})
		if err != nil {
			return nil, err
		}

		call.Inputs = append(call.Inputs, source)
	}

	return call, nil
}

func bindingMap(bindings []model.Binding) (map[string]model.Binding, error) {
	indexed := map[string]model.Binding{}

	for _, binding := range bindings {
		key := typeKey(binding.Interface)
		if existing, exists := indexed[key]; exists {
			return nil, duplicateBindingError(key, existing, binding)
		}

		indexed[key] = binding
	}

	return indexed, nil
}

func providerMap(providers []model.Provider) (map[string][]*model.Provider, error) {
	indexed := map[string][]*model.Provider{}

	for index := range providers {
		provider := &providers[index]
		key := typeKey(provider.Output)
		indexed[key] = append(indexed[key], provider)
	}

	for key, group := range indexed {
		if len(group) <= 1 {
			continue
		}

		return nil, providerSelectionError(fmt.Sprintf("multiple providers for %s", key), nil, group)
	}

	return indexed, nil
}

func selectProvider(spec *model.Spec, overrideBindings map[string]model.Binding, bindings map[string]model.Binding, overrideProviders map[string][]*model.Provider, normalProviders map[string][]*model.Provider, target types.Type, chain []requirement) (*model.Provider, error) {
	mapped := target
	if binding, ok := overrideBindings[typeKey(target)]; ok {
		mapped = binding.Implementation
	} else if binding, ok := bindings[typeKey(target)]; ok {
		mapped = binding.Implementation
	}

	if provider, err := exactProvider(overrideProviders, mapped, chain); err != nil || provider != nil {
		return provider, err
	}

	if provider, err := exactProvider(normalProviders, mapped, chain); err != nil || provider != nil {
		return provider, err
	}

	if provider, err := assignableProvider(spec.Overrides, target, chain, "override"); err != nil || provider != nil {
		return provider, err
	}

	if provider, err := assignableProvider(spec.Providers, target, chain, ""); err != nil || provider != nil {
		return provider, err
	}

	return nil, missingProviderError(target, chain, bindingNote(overrideBindings, bindings, target))
}

func exactProvider(indexed map[string][]*model.Provider, target types.Type, chain []requirement) (*model.Provider, error) {
	candidates := indexed[typeKey(target)]

	switch len(candidates) {
	case 0:
		return nil, nil
	case 1:
		return candidates[0], nil
	default:
		return nil, providerSelectionError(fmt.Sprintf("multiple providers for %s", typeKey(target)), chain, candidates)
	}
}

func assignableProvider(providers []model.Provider, target types.Type, chain []requirement, layer string) (*model.Provider, error) {
	candidates := findProviders(providers, func(provider *model.Provider) bool {
		return types.AssignableTo(provider.Output, target)
	})

	switch len(candidates) {
	case 0:
		return nil, nil
	case 1:
		return candidates[0], nil
	default:
		message := fmt.Sprintf("multiple assignable providers for %s", typeKey(target))
		if layer != "" {
			message = fmt.Sprintf("multiple assignable %s providers for %s", layer, typeKey(target))
		}
		return nil, providerSelectionError(message, chain, candidates)
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

func entryFieldRequirement(entry model.Entry, field model.Field) requirement {
	return requirement{
		ownerKind: "entry field",
		ownerName: entry.Name + "." + field.Name,
		position:  field.Position,
		target:    field.Type,
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

func hookRequirement(fn *model.Function, target types.Type) requirement {
	return requirement{
		ownerKind: "lifecycle hook",
		ownerName: fn.Name,
		position:  fn.Position,
		target:    target,
	}
}

func lifecycleName(lifecycle model.Lifecycle) string {
	switch lifecycle.Kind {
	case model.StartStopLifecycle:
		return "StartStop"
	case model.CloseLifecycle:
		return "Close"
	case model.HookFuncLifecycle:
		return "HookFunc"
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

func bindingNote(overrideBindings map[string]model.Binding, bindings map[string]model.Binding, target types.Type) string {
	if binding, ok := overrideBindings[typeKey(target)]; ok {
		return fmt.Sprintf("override binding at %s maps %s to %s", binding.Position.String(), typeKey(binding.Interface), typeKey(binding.Implementation))
	}

	binding, ok := bindings[typeKey(target)]
	if !ok {
		return ""
	}

	return fmt.Sprintf("binding at %s maps %s to %s", binding.Position.String(), typeKey(binding.Interface), typeKey(binding.Implementation))
}
