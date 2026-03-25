package render

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"unicode"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/resolve"
)

func Go(plan *resolve.Plan) ([]byte, error) {
	aliases := packageAliases(plan)
	variables := variableNames(plan)

	var body strings.Builder
	body.WriteString("package ")
	body.WriteString(plan.Spec.PackageName)
	body.WriteString("\n\n")

	imports := imports(plan)
	if len(imports) > 0 {
		body.WriteString("import (\n")
		for _, path := range imports {
			alias := aliases[path]
			if alias == basePackageName(path) {
				fmt.Fprintf(&body, "\t%q\n", path)
				continue
			}

			fmt.Fprintf(&body, "\t%s %q\n", alias, path)
		}
		body.WriteString(")\n\n")
	}

	body.WriteString("func runBootstrap(ctx context.Context) error {\n")

	for _, step := range plan.Steps {
		name := variables[typeKey(step.Provider.Output)]
		call := renderFunction(plan.Spec.PackagePath, step.Provider.PackagePath, step.Provider.PackageName, step.Provider.Name, aliases)
		args := renderArguments(step.Inputs, variables)

		if step.Provider.HasError {
			fmt.Fprintf(&body, "\t%s, err := %s(%s)\n", name, call, args)
			body.WriteString("\tif err != nil {\n")
			body.WriteString("\t\treturn err\n")
			body.WriteString("\t}\n")
			continue
		}

		fmt.Fprintf(&body, "\t%s := %s(%s)\n", name, call, args)
	}

	for _, lifecycle := range plan.Lifecycles {
		target := variables[typeKey(lifecycle.Source.Provider.Output)]

		switch lifecycle.Spec.Kind {
		case model.StartStopLifecycle:
			startCall, err := renderMethodCall(lifecycle.Source.Provider.Output, target, lifecycle.Spec.Start, true)
			if err != nil {
				return nil, err
			}

			if startCall.HasError {
				fmt.Fprintf(&body, "\tif err := %s; err != nil {\n", startCall.Expr)
				body.WriteString("\t\treturn err\n")
				body.WriteString("\t}\n")
			} else {
				fmt.Fprintf(&body, "\t%s\n", startCall.Expr)
			}

			stopCall, err := renderMethodCall(lifecycle.Source.Provider.Output, target, lifecycle.Spec.Stop, true)
			if err != nil {
				return nil, err
			}

			if stopCall.HasError {
				fmt.Fprintf(&body, "\tdefer func() { _ = %s }()\n", stopCall.Expr)
			} else {
				fmt.Fprintf(&body, "\tdefer %s\n", stopCall.Expr)
			}
		case model.CloseLifecycle:
			closeCall, err := renderMethodCall(lifecycle.Source.Provider.Output, target, "Close", false)
			if err != nil {
				return nil, err
			}

			if closeCall.HasError {
				fmt.Fprintf(&body, "\tdefer func() { _ = %s }()\n", closeCall.Expr)
			} else {
				fmt.Fprintf(&body, "\tdefer %s\n", closeCall.Expr)
			}
		}
	}

	entryCall := renderFunction(plan.Spec.PackagePath, plan.Spec.Entry.PackagePath, plan.Spec.Entry.PackageName, plan.Spec.Entry.Name, aliases)
	entryArgs := renderArguments(plan.Entry, variables)

	if plan.Spec.Entry.ReturnsError {
		fmt.Fprintf(&body, "\treturn %s(%s)\n", entryCall, entryArgs)
	} else {
		fmt.Fprintf(&body, "\t%s(%s)\n", entryCall, entryArgs)
		body.WriteString("\treturn nil\n")
	}

	body.WriteString("}\n")

	formatted, err := format.Source([]byte(body.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated code: %w\n%s", err, body.String())
	}

	return formatted, nil
}

type methodCall struct {
	Expr     string
	HasError bool
}

func renderMethodCall(targetType types.Type, target string, method string, allowContext bool) (methodCall, error) {
	fn, ok := lookupMethod(targetType, method)
	if !ok {
		return methodCall{}, fmt.Errorf("method %s not found on %s", method, typeKey(targetType))
	}

	signature, ok := fn.Type().Underlying().(*types.Signature)
	if !ok {
		return methodCall{}, fmt.Errorf("%s is not a method", method)
	}

	args := ""
	switch signature.Params().Len() {
	case 0:
	case 1:
		if !allowContext || typeKey(signature.Params().At(0).Type()) != "context.Context" {
			return methodCall{}, fmt.Errorf("method %s has unsupported parameters", method)
		}

		args = "ctx"
	default:
		return methodCall{}, fmt.Errorf("method %s has unsupported parameters", method)
	}

	switch signature.Results().Len() {
	case 0:
	case 1:
		if !isErrorType(signature.Results().At(0).Type()) {
			return methodCall{}, fmt.Errorf("method %s must return nothing or error", method)
		}
	default:
		return methodCall{}, fmt.Errorf("method %s must return nothing or error", method)
	}

	return methodCall{
		Expr:     fmt.Sprintf("%s.%s(%s)", target, method, args),
		HasError: signature.Results().Len() == 1,
	}, nil
}

func lookupMethod(targetType types.Type, name string) (*types.Func, bool) {
	sets := []*types.MethodSet{types.NewMethodSet(targetType)}
	if _, ok := targetType.(*types.Pointer); !ok {
		sets = append(sets, types.NewMethodSet(types.NewPointer(targetType)))
	}

	for _, set := range sets {
		for index := range set.Len() {
			selection := set.At(index)
			if selection.Obj().Name() != name {
				continue
			}

			fn, ok := selection.Obj().(*types.Func)
			if ok {
				return fn, true
			}
		}
	}

	return nil, false
}

func imports(plan *resolve.Plan) []string {
	set := map[string]struct{}{
		"context": {},
	}

	for _, step := range plan.Steps {
		addImport(set, plan.Spec.PackagePath, step.Provider.PackagePath)
	}

	addImport(set, plan.Spec.PackagePath, plan.Spec.Entry.PackagePath)

	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}

	sort.Strings(paths)
	return paths
}

func addImport(set map[string]struct{}, current string, path string) {
	if path == "" || path == current {
		return
	}

	set[path] = struct{}{}
}

func packageAliases(plan *resolve.Plan) map[string]string {
	paths := map[string]string{
		"context": "context",
	}

	for _, step := range plan.Steps {
		if step.Provider.PackagePath != "" && step.Provider.PackagePath != plan.Spec.PackagePath {
			paths[step.Provider.PackagePath] = step.Provider.PackageName
		}
	}

	if plan.Spec.Entry.PackagePath != "" && plan.Spec.Entry.PackagePath != plan.Spec.PackagePath {
		paths[plan.Spec.Entry.PackagePath] = plan.Spec.Entry.PackageName
	}

	aliases := map[string]string{}
	used := map[string]int{}

	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}

	sort.Strings(ordered)

	for _, path := range ordered {
		base := sanitizeIdentifier(paths[path])
		if base == "" {
			base = sanitizeIdentifier(basePackageName(path))
		}
		if base == "" {
			base = "pkg"
		}

		alias := base
		if used[alias] > 0 {
			alias = fmt.Sprintf("%s%d", alias, used[alias]+1)
		}

		used[base]++
		aliases[path] = alias
	}

	return aliases
}

func variableNames(plan *resolve.Plan) map[string]string {
	names := map[string]string{}
	used := map[string]int{}

	for _, step := range plan.Steps {
		base := valueName(step.Provider.Output)
		alias := base
		if used[base] > 0 {
			alias = fmt.Sprintf("%s%d", base, used[base]+1)
		}

		used[base]++
		names[typeKey(step.Provider.Output)] = alias
	}

	return names
}

func renderFunction(currentPath string, packagePath string, packageName string, name string, aliases map[string]string) string {
	if packagePath == "" || packagePath == currentPath {
		return name
	}

	return aliases[packagePath] + "." + name
}

func renderArguments(inputs []resolve.Source, variables map[string]string) string {
	args := make([]string, 0, len(inputs))

	for _, input := range inputs {
		switch input.Kind {
		case resolve.ContextSource:
			args = append(args, "ctx")
		case resolve.ProviderSource:
			args = append(args, variables[typeKey(input.Provider.Output)])
		}
	}

	return strings.Join(args, ", ")
}

func valueName(typ types.Type) string {
	switch value := typ.(type) {
	case *types.Pointer:
		return valueName(value.Elem())
	case *types.Named:
		return sanitizeIdentifier(lowerFirst(value.Obj().Name()))
	case *types.Slice:
		return sanitizeIdentifier(valueName(value.Elem()) + "List")
	case *types.Array:
		return sanitizeIdentifier(valueName(value.Elem()) + "List")
	case *types.Map:
		return "valueMap"
	case *types.Basic:
		return sanitizeIdentifier(value.Name() + "Value")
	default:
		return "value"
	}
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func sanitizeIdentifier(value string) string {
	if value == "" {
		return ""
	}

	var builder bytes.Buffer
	for index, r := range value {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) && index > 0 {
			builder.WriteRune(r)
			continue
		}

		if unicode.IsDigit(r) && index == 0 {
			builder.WriteRune('_')
			builder.WriteRune(r)
		}
	}

	result := builder.String()
	if result == "" {
		return ""
	}

	if token.Lookup(result).IsKeyword() {
		return result + "Value"
	}

	return result
}

func basePackageName(path string) string {
	if index := strings.LastIndex(path, "/"); index >= 0 {
		return path[index+1:]
	}

	return path
}

func isErrorType(typ types.Type) bool {
	return typeKey(typ) == "error"
}

func typeKey(typ types.Type) string {
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}

		return pkg.Path()
	})
}
