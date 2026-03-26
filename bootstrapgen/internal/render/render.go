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
	allocator := newIdentifierAllocator(plan.Spec.PackageName, aliases)
	variables, entryVars := variableNames(plan, allocator)

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

	for index, arg := range plan.Entry {
		if len(arg.Fields) == 0 {
			continue
		}

		name := entryVars[index]
		fmt.Fprintf(&body, "\t%s := %s{\n", name, renderType(arg.Type, plan.Spec.PackagePath, aliases))
		for _, field := range arg.Fields {
			fmt.Fprintf(&body, "\t\t%s: %s,\n", field.Name, renderSource(field.Source, variables))
		}
		body.WriteString("\t}\n")
	}

	for _, lifecycle := range plan.Lifecycles {
		switch lifecycle.Spec.Kind {
		case model.StartStopLifecycle:
			target := renderSource(lifecycle.Source, variables)
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
			target := renderSource(lifecycle.Source, variables)
			closeCall, err := renderMethodCall(lifecycle.Source.Provider.Output, target, "Close", false)
			if err != nil {
				return nil, err
			}

			if closeCall.HasError {
				fmt.Fprintf(&body, "\tdefer func() { _ = %s }()\n", closeCall.Expr)
			} else {
				fmt.Fprintf(&body, "\tdefer %s\n", closeCall.Expr)
			}
		case model.HookFuncLifecycle:
			if lifecycle.Start != nil {
				call := renderHookCall(plan.Spec.PackagePath, lifecycle.Start, aliases, variables)
				if lifecycle.Start.Func.ReturnsError {
					fmt.Fprintf(&body, "\tif err := %s; err != nil {\n", call)
					body.WriteString("\t\treturn err\n")
					body.WriteString("\t}\n")
				} else {
					fmt.Fprintf(&body, "\t%s\n", call)
				}
			}

			if lifecycle.Stop != nil {
				call := renderHookCall(plan.Spec.PackagePath, lifecycle.Stop, aliases, variables)
				if lifecycle.Stop.Func.ReturnsError {
					fmt.Fprintf(&body, "\tdefer func() { _ = %s }()\n", call)
				} else {
					fmt.Fprintf(&body, "\tdefer %s\n", call)
				}
			}
		}
	}

	entryCall := renderFunction(plan.Spec.PackagePath, plan.Spec.Entry.PackagePath, plan.Spec.Entry.PackageName, plan.Spec.Entry.Name, aliases)
	entryArgs := renderEntryArguments(plan.Entry, variables, entryVars)

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

	for _, arg := range plan.Entry {
		if len(arg.Fields) > 0 {
			collectTypeImports(set, plan.Spec.PackagePath, arg.Type)
		}
	}

	for _, lifecycle := range plan.Lifecycles {
		if lifecycle.Start != nil {
			addImport(set, plan.Spec.PackagePath, lifecycle.Start.Func.PackagePath)
		}
		if lifecycle.Stop != nil {
			addImport(set, plan.Spec.PackagePath, lifecycle.Stop.Func.PackagePath)
		}
	}

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

func collectTypeImports(set map[string]struct{}, current string, typ types.Type) {
	switch value := typ.(type) {
	case *types.Pointer:
		collectTypeImports(set, current, value.Elem())
	case *types.Named:
		addImport(set, current, packagePath(value.Obj().Pkg()))
	case *types.Slice:
		collectTypeImports(set, current, value.Elem())
	case *types.Array:
		collectTypeImports(set, current, value.Elem())
	case *types.Map:
		collectTypeImports(set, current, value.Key())
		collectTypeImports(set, current, value.Elem())
	case *types.Struct:
		for index := range value.NumFields() {
			collectTypeImports(set, current, value.Field(index).Type())
		}
	}
}

func packageAliases(plan *resolve.Plan) map[string]string {
	paths := map[string]string{
		"context": "context",
	}

	for _, step := range plan.Steps {
		addPackageName(paths, step.Provider.PackagePath, step.Provider.PackageName)
	}

	addPackageName(paths, plan.Spec.Entry.PackagePath, plan.Spec.Entry.PackageName)

	for _, arg := range plan.Entry {
		if len(arg.Fields) > 0 {
			collectTypePackageNames(paths, arg.Type)
		}
	}

	for _, lifecycle := range plan.Lifecycles {
		if lifecycle.Start != nil {
			addPackageName(paths, lifecycle.Start.Func.PackagePath, lifecycle.Start.Func.PackageName)
		}
		if lifecycle.Stop != nil {
			addPackageName(paths, lifecycle.Stop.Func.PackagePath, lifecycle.Stop.Func.PackageName)
		}
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
		if used[base] > 0 {
			alias = fmt.Sprintf("%s%d", base, used[base]+1)
		}

		used[base]++
		aliases[path] = alias
	}

	return aliases
}

func addPackageName(paths map[string]string, path string, name string) {
	if path == "" {
		return
	}

	if _, ok := paths[path]; ok {
		return
	}

	paths[path] = name
}

func collectTypePackageNames(paths map[string]string, typ types.Type) {
	switch value := typ.(type) {
	case *types.Pointer:
		collectTypePackageNames(paths, value.Elem())
	case *types.Named:
		addPackageName(paths, packagePath(value.Obj().Pkg()), packageName(value.Obj().Pkg()))
	case *types.Slice:
		collectTypePackageNames(paths, value.Elem())
	case *types.Array:
		collectTypePackageNames(paths, value.Elem())
	case *types.Map:
		collectTypePackageNames(paths, value.Key())
		collectTypePackageNames(paths, value.Elem())
	case *types.Struct:
		for index := range value.NumFields() {
			collectTypePackageNames(paths, value.Field(index).Type())
		}
	}
}

func variableNames(plan *resolve.Plan, allocator *identifierAllocator) (map[string]string, map[int]string) {
	names := map[string]string{}
	entryVars := map[int]string{}

	for _, step := range plan.Steps {
		names[typeKey(step.Provider.Output)] = allocator.Allocate(valueName(plan.Spec.PackagePath, step.Provider.Output))
	}

	for index, arg := range plan.Entry {
		if len(arg.Fields) == 0 {
			continue
		}

		entryVars[index] = allocator.Allocate(valueName(plan.Spec.PackagePath, arg.Type))
	}

	return names, entryVars
}

func renderFunction(currentPath string, packagePath string, packageName string, name string, aliases map[string]string) string {
	if packagePath == "" || packagePath == currentPath {
		return name
	}

	return aliases[packagePath] + "." + name
}

func renderHookCall(currentPath string, hook *resolve.HookCall, aliases map[string]string, variables map[string]string) string {
	call := renderFunction(currentPath, hook.Func.PackagePath, hook.Func.PackageName, hook.Func.Name, aliases)
	return fmt.Sprintf("%s(%s)", call, renderArguments(hook.Inputs, variables))
}

func renderEntryArguments(args []resolve.EntryArg, variables map[string]string, entryVars map[int]string) string {
	rendered := make([]string, 0, len(args))

	for index, arg := range args {
		if len(arg.Fields) == 0 {
			rendered = append(rendered, renderSource(arg.Source, variables))
			continue
		}

		rendered = append(rendered, entryVars[index])
	}

	return strings.Join(rendered, ", ")
}

func renderArguments(inputs []resolve.Source, variables map[string]string) string {
	args := make([]string, 0, len(inputs))

	for _, input := range inputs {
		args = append(args, renderSource(input, variables))
	}

	return strings.Join(args, ", ")
}

func renderSource(input resolve.Source, variables map[string]string) string {
	switch input.Kind {
	case resolve.ContextSource:
		return "ctx"
	case resolve.ProviderSource:
		return variables[typeKey(input.Provider.Output)]
	default:
		return ""
	}
}

func renderType(typ types.Type, current string, aliases map[string]string) string {
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil || pkg.Path() == current {
			return ""
		}

		return aliases[pkg.Path()]
	})
}

type identifierAllocator struct {
	used map[string]int
}

func newIdentifierAllocator(packageName string, aliases map[string]string) *identifierAllocator {
	used := map[string]int{}

	for _, keyword := range reservedIdentifiers() {
		used[keyword] = 1
	}

	for _, fixed := range []string{"ctx", "err", packageName} {
		if fixed != "" {
			used[fixed] = 1
		}
	}

	for _, alias := range aliases {
		if alias != "" {
			used[alias] = 1
		}
	}

	return &identifierAllocator{used: used}
}

func (allocator *identifierAllocator) Allocate(base string) string {
	base = sanitizeIdentifier(base)
	if base == "" {
		base = "value"
	}

	if allocator.used[base] == 0 {
		allocator.used[base] = 1
		return base
	}

	for index := allocator.used[base] + 1; ; index++ {
		candidate := fmt.Sprintf("%s%d", base, index)
		if allocator.used[candidate] == 0 {
			allocator.used[base] = index
			allocator.used[candidate] = 1
			return candidate
		}
	}
}

func reservedIdentifiers() []string {
	return []string{
		"any",
		"append",
		"bool",
		"byte",
		"cap",
		"case",
		"chan",
		"clear",
		"close",
		"comparable",
		"complex",
		"complex128",
		"complex64",
		"const",
		"continue",
		"copy",
		"default",
		"delete",
		"else",
		"error",
		"fallthrough",
		"false",
		"float32",
		"float64",
		"for",
		"func",
		"go",
		"goto",
		"if",
		"imag",
		"import",
		"int",
		"int16",
		"int32",
		"int64",
		"int8",
		"interface",
		"iota",
		"len",
		"make",
		"map",
		"max",
		"min",
		"new",
		"nil",
		"package",
		"panic",
		"print",
		"println",
		"range",
		"real",
		"recover",
		"return",
		"rune",
		"select",
		"string",
		"struct",
		"switch",
		"true",
		"type",
		"uint",
		"uint16",
		"uint32",
		"uint64",
		"uint8",
		"uintptr",
		"var",
	}
}

func valueName(currentPath string, typ types.Type) string {
	switch value := typ.(type) {
	case *types.Pointer:
		return valueName(currentPath, value.Elem())
	case *types.Named:
		name := value.Obj().Name()
		if value.Obj().Pkg() == nil || value.Obj().Pkg().Path() == currentPath {
			return sanitizeIdentifier(lowerFirst(name))
		}

		prefix := sanitizeIdentifier(lowerFirst(value.Obj().Pkg().Name()))
		if prefix == "" {
			prefix = sanitizeIdentifier(lowerFirst(basePackageName(value.Obj().Pkg().Path())))
		}
		if prefix == sanitizeIdentifier(lowerFirst(name)) {
			return prefix
		}
		return sanitizeIdentifier(prefix + upperFirst(name))
	case *types.Slice:
		return sanitizeIdentifier(valueName(currentPath, value.Elem()) + "List")
	case *types.Array:
		return sanitizeIdentifier(valueName(currentPath, value.Elem()) + "List")
	case *types.Map:
		return "valueMap"
	case *types.Basic:
		return sanitizeIdentifier(value.Name() + "Value")
	case *types.Struct:
		return "params"
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

func upperFirst(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
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

func packageName(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}

	return pkg.Name()
}

func packagePath(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}

	return pkg.Path()
}
