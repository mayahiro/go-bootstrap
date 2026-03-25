package scan

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"strconv"

	"golang.org/x/tools/go/packages"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
)

const bootstrapPackagePath = "github.com/mayahiro/go-bootstrap/bootstrap"

func Package(pkg *packages.Package, fset *token.FileSet) (*model.Spec, error) {
	var spec *model.Spec

	for _, file := range pkg.Syntax {
		var scanErr error

		ast.Inspect(file, func(node ast.Node) bool {
			if scanErr != nil {
				return false
			}

			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			kind, ok := bootstrapCall(pkg.TypesInfo, call.Fun, "Server", "CLI")
			if !ok {
				return true
			}

			if spec != nil {
				scanErr = nodeError(fset, call, "multiple bootstrap specs found in %s", pkg.PkgPath)
				return false
			}

			spec, scanErr = parseSpec(pkg, fset, kind, call)
			return false
		})

		if scanErr != nil {
			return nil, scanErr
		}
	}

	if spec == nil {
		return nil, fmt.Errorf("bootstrap spec not found in %s", pkg.PkgPath)
	}

	if len(pkg.GoFiles) == 0 {
		return nil, fmt.Errorf("package %s has no go files", pkg.PkgPath)
	}

	spec.PackageName = pkg.Name
	spec.PackagePath = pkg.PkgPath
	spec.Directory = filepath.Dir(pkg.GoFiles[0])

	return spec, nil
}

func parseSpec(pkg *packages.Package, fset *token.FileSet, kind string, call *ast.CallExpr) (*model.Spec, error) {
	if len(call.Args) == 0 {
		return nil, nodeError(fset, call, "%s requires a name", kind)
	}

	name, err := stringLiteral(call.Args[0])
	if err != nil {
		return nil, nodeMessage(fset, call.Args[0], err.Error())
	}

	spec := &model.Spec{
		Kind:     kind,
		Name:     name,
		Position: positionAt(fset, call.Pos()),
	}

	for _, arg := range call.Args[1:] {
		option, ok := arg.(*ast.CallExpr)
		if !ok {
			return nil, nodeError(fset, arg, "bootstrap option must be a call")
		}

		optionName, ok := bootstrapCall(pkg.TypesInfo, option.Fun, "Provide", "Bind", "Entry", "Lifecycle")
		if !ok {
			return nil, nodeError(fset, option, "unsupported bootstrap option")
		}

		switch optionName {
		case "Provide":
			for _, ctor := range option.Args {
				provider, err := parseProvider(pkg, fset, ctor)
				if err != nil {
					return nil, err
				}

				spec.Providers = append(spec.Providers, provider)
			}
		case "Bind":
			if len(option.Args) != 2 {
				return nil, nodeError(fset, option, "Bind requires 2 arguments")
			}

			iface, err := parseBindingType(pkg, option.Args[0])
			if err != nil {
				return nil, nodeMessage(fset, option.Args[0], err.Error())
			}

			impl, err := parseBindingType(pkg, option.Args[1])
			if err != nil {
				return nil, nodeMessage(fset, option.Args[1], err.Error())
			}

			spec.Bindings = append(spec.Bindings, model.Binding{
				Interface:      iface,
				Implementation: impl,
				Position:       positionAt(fset, option.Pos()),
			})
		case "Entry":
			if len(option.Args) != 1 {
				return nil, nodeError(fset, option, "Entry requires 1 argument")
			}

			entry, err := parseEntry(pkg, fset, option.Args[0])
			if err != nil {
				return nil, err
			}

			spec.Entry = entry
		case "Lifecycle":
			for _, hook := range option.Args {
				lifecycle, err := parseLifecycle(pkg, fset, hook)
				if err != nil {
					return nil, err
				}

				spec.Lifecycles = append(spec.Lifecycles, lifecycle)
			}
		}
	}

	if spec.Entry.Name == "" {
		return nil, nodeError(fset, call, "Entry is required")
	}

	return spec, nil
}

func parseProvider(pkg *packages.Package, fset *token.FileSet, expr ast.Expr) (model.Provider, error) {
	fn, sig, err := parseFunction(pkg, expr)
	if err != nil {
		return model.Provider{}, nodeMessage(fset, expr, err.Error())
	}

	results := sig.Results()
	if results.Len() == 0 || results.Len() > 2 {
		return model.Provider{}, nodeError(fset, expr, "provider %s must return T or (T, error)", fn.Name())
	}

	output := results.At(0).Type()
	hasError := false

	if results.Len() == 2 {
		if !isErrorType(results.At(1).Type()) {
			return model.Provider{}, nodeError(fset, expr, "provider %s second result must be error", fn.Name())
		}

		hasError = true
	}

	return model.Provider{
		Name:        fn.Name(),
		Position:    positionAt(fset, expr.Pos()),
		PackageName: packageName(fn.Pkg()),
		PackagePath: packagePath(fn.Pkg()),
		Inputs:      tupleTypes(sig.Params()),
		Output:      output,
		HasError:    hasError,
	}, nil
}

func parseEntry(pkg *packages.Package, fset *token.FileSet, expr ast.Expr) (model.Entry, error) {
	fn, sig, err := parseFunction(pkg, expr)
	if err != nil {
		return model.Entry{}, nodeMessage(fset, expr, err.Error())
	}

	results := sig.Results()
	returnsError := false

	switch results.Len() {
	case 0:
	case 1:
		if !isErrorType(results.At(0).Type()) {
			return model.Entry{}, nodeError(fset, expr, "entry %s result must be error", fn.Name())
		}

		returnsError = true
	default:
		return model.Entry{}, nodeError(fset, expr, "entry %s must return nothing or error", fn.Name())
	}

	return model.Entry{
		Name:         fn.Name(),
		Position:     positionAt(fset, expr.Pos()),
		PackageName:  packageName(fn.Pkg()),
		PackagePath:  packagePath(fn.Pkg()),
		Inputs:       tupleTypes(sig.Params()),
		ReturnsError: returnsError,
	}, nil
}

func parseLifecycle(pkg *packages.Package, fset *token.FileSet, expr ast.Expr) (model.Lifecycle, error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return model.Lifecycle{}, nodeError(fset, expr, "lifecycle hook must be a call")
	}

	name, ok := bootstrapCall(pkg.TypesInfo, call.Fun, "Close", "StartStop")
	if !ok {
		return model.Lifecycle{}, nodeError(fset, call, "unsupported lifecycle hook")
	}

	switch name {
	case "Close":
		if len(call.Args) != 1 {
			return model.Lifecycle{}, nodeError(fset, call, "Close requires 1 argument")
		}

		target, err := parseBindingType(pkg, call.Args[0])
		if err != nil {
			return model.Lifecycle{}, nodeMessage(fset, call.Args[0], err.Error())
		}

		return model.Lifecycle{
			Kind:     model.CloseLifecycle,
			Target:   target,
			Position: positionAt(fset, call.Pos()),
		}, nil
	case "StartStop":
		if len(call.Args) != 3 {
			return model.Lifecycle{}, nodeError(fset, call, "StartStop requires target, start, stop")
		}

		target, err := parseBindingType(pkg, call.Args[0])
		if err != nil {
			return model.Lifecycle{}, nodeMessage(fset, call.Args[0], err.Error())
		}

		start, err := stringLiteral(call.Args[1])
		if err != nil {
			return model.Lifecycle{}, nodeMessage(fset, call.Args[1], err.Error())
		}

		stop, err := stringLiteral(call.Args[2])
		if err != nil {
			return model.Lifecycle{}, nodeMessage(fset, call.Args[2], err.Error())
		}

		return model.Lifecycle{
			Kind:     model.StartStopLifecycle,
			Target:   target,
			Start:    start,
			Stop:     stop,
			Position: positionAt(fset, call.Pos()),
		}, nil
	default:
		return model.Lifecycle{}, fmt.Errorf("unsupported lifecycle hook")
	}
}

func parseFunction(pkg *packages.Package, expr ast.Expr) (*types.Func, *types.Signature, error) {
	obj := functionObject(pkg.TypesInfo, expr)
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, nil, fmt.Errorf("expected function reference")
	}

	sig, ok := pkg.TypesInfo.TypeOf(expr).Underlying().(*types.Signature)
	if !ok {
		return nil, nil, fmt.Errorf("%s is not a function", fn.Name())
	}

	return fn, sig, nil
}

func parseBindingType(pkg *packages.Package, expr ast.Expr) (types.Type, error) {
	typ := pkg.TypesInfo.TypeOf(expr)
	if typ == nil {
		return nil, fmt.Errorf("failed to resolve type")
	}

	if pointer, ok := typ.(*types.Pointer); ok {
		if _, ok := pointer.Elem().Underlying().(*types.Interface); ok {
			return pointer.Elem(), nil
		}

		return pointer, nil
	}

	return typ, nil
}

func functionObject(info *types.Info, expr ast.Expr) types.Object {
	switch value := expr.(type) {
	case *ast.Ident:
		return info.ObjectOf(value)
	case *ast.SelectorExpr:
		return info.ObjectOf(value.Sel)
	default:
		return nil
	}
}

func bootstrapCall(info *types.Info, expr ast.Expr, names ...string) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	pkgName, ok := info.ObjectOf(pkgIdent).(*types.PkgName)
	if !ok {
		return "", false
	}

	if pkgName.Imported().Path() != bootstrapPackagePath {
		return "", false
	}

	for _, name := range names {
		if selector.Sel.Name == name {
			return name, true
		}
	}

	return "", false
}

func tupleTypes(tuple *types.Tuple) []types.Type {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}

	typesList := make([]types.Type, 0, tuple.Len())
	for index := range tuple.Len() {
		typesList = append(typesList, tuple.At(index).Type())
	}

	return typesList
}

func stringLiteral(expr ast.Expr) (string, error) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", fmt.Errorf("expected string literal")
	}

	return strconv.Unquote(lit.Value)
}

func isErrorType(typ types.Type) bool {
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}

		return pkg.Path()
	}) == "error"
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

func nodeError(fset *token.FileSet, node ast.Node, format string, args ...any) error {
	return fmt.Errorf("%s: %s", positionAt(fset, node.Pos()).String(), fmt.Sprintf(format, args...))
}

func nodeMessage(fset *token.FileSet, node ast.Node, message string) error {
	return fmt.Errorf("%s: %s", positionAt(fset, node.Pos()).String(), message)
}

func positionAt(fset *token.FileSet, pos token.Pos) model.Position {
	if fset == nil {
		return model.Position{}
	}

	position := fset.Position(pos)
	if !position.IsValid() {
		return model.Position{}
	}

	return model.Position{
		File:   path.Base(position.Filename),
		Line:   position.Line,
		Column: position.Column,
	}
}
