package scan

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
)

const bootstrapPackagePath = "github.com/mayahiro/go-bootstrap/bootstrap"

type scanner struct {
	fset     *token.FileSet
	packages map[string]*packages.Package
	indexes  map[string]*packageIndex
}

type packageIndex struct {
	values map[types.Object]ast.Expr
}

type moduleRef struct {
	Key      string
	Name     string
	Position model.Position
}

func Package(pkg *packages.Package, fset *token.FileSet) (*model.Spec, error) {
	scan := &scanner{
		fset:     fset,
		packages: collectPackages(pkg),
		indexes:  map[string]*packageIndex{},
	}

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

			spec, scanErr = scan.parseSpec(pkg, kind, call)
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

func (scan *scanner) parseSpec(pkg *packages.Package, kind string, call *ast.CallExpr) (*model.Spec, error) {
	if len(call.Args) == 0 {
		return nil, nodeError(scan.fset, call, "%s requires a name", kind)
	}

	name, err := stringLiteral(call.Args[0])
	if err != nil {
		return nil, nodeMessage(scan.fset, call.Args[0], err.Error())
	}

	spec := &model.Spec{
		Kind:     kind,
		Name:     name,
		Position: positionAt(scan.fset, call.Pos()),
	}

	if err := scan.applyOptions(spec, pkg, call.Args[1:], nil, false); err != nil {
		return nil, err
	}

	if spec.Entry.Name == "" {
		return nil, nodeError(scan.fset, call, "Entry is required")
	}

	return spec, nil
}

func (scan *scanner) applyOptions(spec *model.Spec, pkg *packages.Package, args []ast.Expr, stack []moduleRef, inModule bool) error {
	for _, arg := range args {
		option, ok := arg.(*ast.CallExpr)
		if !ok {
			return nodeError(scan.fset, arg, "bootstrap option must be a call")
		}

		optionName, ok := bootstrapCall(pkg.TypesInfo, option.Fun, "Provide", "Bind", "Entry", "Lifecycle", "Include")
		if !ok {
			return nodeError(scan.fset, option, "unsupported bootstrap option")
		}

		switch optionName {
		case "Provide":
			for _, ctor := range option.Args {
				provider, err := parseProvider(pkg, scan.fset, ctor)
				if err != nil {
					return err
				}

				spec.Providers = append(spec.Providers, provider)
			}
		case "Bind":
			if len(option.Args) != 2 {
				return nodeError(scan.fset, option, "Bind requires 2 arguments")
			}

			iface, err := parseBindingType(pkg, option.Args[0])
			if err != nil {
				return nodeMessage(scan.fset, option.Args[0], err.Error())
			}

			impl, err := parseBindingType(pkg, option.Args[1])
			if err != nil {
				return nodeMessage(scan.fset, option.Args[1], err.Error())
			}

			spec.Bindings = append(spec.Bindings, model.Binding{
				Interface:      iface,
				Implementation: impl,
				Position:       positionAt(scan.fset, option.Pos()),
			})
		case "Entry":
			if inModule {
				return nodeError(scan.fset, option, "Entry is not allowed inside Module")
			}

			if len(option.Args) != 1 {
				return nodeError(scan.fset, option, "Entry requires 1 argument")
			}

			entry, err := parseEntry(pkg, scan.fset, option.Args[0])
			if err != nil {
				return err
			}

			spec.Entry = entry
		case "Lifecycle":
			for _, hook := range option.Args {
				lifecycle, err := parseLifecycle(pkg, scan.fset, hook)
				if err != nil {
					return err
				}

				spec.Lifecycles = append(spec.Lifecycles, lifecycle)
			}
		case "Include":
			for _, moduleExpr := range option.Args {
				if err := scan.includeModule(spec, pkg, moduleExpr, stack); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (scan *scanner) includeModule(spec *model.Spec, pkg *packages.Package, expr ast.Expr, stack []moduleRef) error {
	modulePkg, call, ref, err := scan.resolveModule(pkg, expr)
	if err != nil {
		return err
	}

	if ref.Key != "" {
		for _, current := range stack {
			if current.Key == ref.Key {
				chain := make([]string, 0, len(stack)+1)
				for _, entry := range stack {
					chain = append(chain, fmt.Sprintf("%s at %s", entry.Name, entry.Position.String()))
				}
				chain = append(chain, fmt.Sprintf("%s at %s", ref.Name, ref.Position.String()))
				return nodeError(scan.fset, expr, "module include cycle detected: %s", strings.Join(chain, " -> "))
			}
		}

		stack = append(stack, ref)
	}

	return scan.parseModuleCall(spec, modulePkg, call, stack)
}

func (scan *scanner) resolveModule(pkg *packages.Package, expr ast.Expr) (*packages.Package, *ast.CallExpr, moduleRef, error) {
	expr = unwrap(expr)

	call, ok := expr.(*ast.CallExpr)
	if ok {
		if _, match := bootstrapCall(pkg.TypesInfo, call.Fun, "Module"); match {
			return pkg, call, moduleRef{}, nil
		}
	}

	obj := moduleObject(pkg.TypesInfo, expr)
	if obj == nil {
		return nil, nil, moduleRef{}, nodeError(scan.fset, expr, "module reference must be a bootstrap.Module value")
	}

	modulePkg := scan.packages[packagePath(obj.Pkg())]
	if modulePkg == nil {
		return nil, nil, moduleRef{}, nodeError(scan.fset, expr, "failed to load package for module %s", obj.Name())
	}

	index := scan.packageIndex(modulePkg)
	initExpr, ok := index.values[obj]
	if !ok {
		return nil, nil, moduleRef{}, nodeError(scan.fset, expr, "module %s must be declared with bootstrap.Module(...)", obj.Name())
	}

	call, ok = unwrap(initExpr).(*ast.CallExpr)
	if !ok {
		return nil, nil, moduleRef{}, nodeError(scan.fset, initExpr, "module %s must be declared with bootstrap.Module(...)", obj.Name())
	}

	if _, match := bootstrapCall(modulePkg.TypesInfo, call.Fun, "Module"); !match {
		return nil, nil, moduleRef{}, nodeError(scan.fset, initExpr, "module %s must be declared with bootstrap.Module(...)", obj.Name())
	}

	return modulePkg, call, moduleRef{
		Key:      moduleKey(obj),
		Name:     obj.Name(),
		Position: positionAt(scan.fset, obj.Pos()),
	}, nil
}

func (scan *scanner) parseModuleCall(spec *model.Spec, pkg *packages.Package, call *ast.CallExpr, stack []moduleRef) error {
	return scan.applyOptions(spec, pkg, call.Args, stack, true)
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

	inputs := make([]model.EntryInput, 0, sig.Params().Len())
	paramsStructCount := 0
	for index := range sig.Params().Len() {
		param := sig.Params().At(index)
		input := model.EntryInput{
			Type:     param.Type(),
			Position: positionAt(fset, param.Pos()),
		}

		fields, ok := parseParamsFields(fset, param.Type())
		if ok {
			paramsStructCount++
			if paramsStructCount > 1 {
				return model.Entry{}, nodeError(fset, expr, "entry %s can have at most one parameter struct embedding bootstrap.In", fn.Name())
			}
			input.Fields = fields
		}

		inputs = append(inputs, input)
	}

	return model.Entry{
		Name:         fn.Name(),
		Position:     positionAt(fset, expr.Pos()),
		PackageName:  packageName(fn.Pkg()),
		PackagePath:  packagePath(fn.Pkg()),
		Inputs:       inputs,
		ReturnsError: returnsError,
	}, nil
}

func parseParamsFields(fset *token.FileSet, typ types.Type) ([]model.Field, bool) {
	strct, ok := structType(typ)
	if !ok {
		return nil, false
	}

	hasMarker := false
	fields := make([]model.Field, 0, strct.NumFields())
	for index := range strct.NumFields() {
		field := strct.Field(index)
		if field.Embedded() && isBootstrapInType(field.Type()) {
			hasMarker = true
			continue
		}

		fields = append(fields, model.Field{
			Name:     field.Name(),
			Type:     field.Type(),
			Position: positionAt(fset, field.Pos()),
		})
	}

	return fields, hasMarker
}

func parseLifecycle(pkg *packages.Package, fset *token.FileSet, expr ast.Expr) (model.Lifecycle, error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return model.Lifecycle{}, nodeError(fset, expr, "lifecycle hook must be a call")
	}

	name, ok := bootstrapCall(pkg.TypesInfo, call.Fun, "Close", "StartStop", "HookFunc")
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
	case "HookFunc":
		if len(call.Args) != 2 {
			return model.Lifecycle{}, nodeError(fset, call, "HookFunc requires start and stop")
		}

		start, err := parseHookFunction(pkg, fset, call.Args[0])
		if err != nil {
			return model.Lifecycle{}, err
		}

		stop, err := parseHookFunction(pkg, fset, call.Args[1])
		if err != nil {
			return model.Lifecycle{}, err
		}

		if start == nil && stop == nil {
			return model.Lifecycle{}, nodeError(fset, call, "HookFunc requires start or stop")
		}

		return model.Lifecycle{
			Kind:     model.HookFuncLifecycle,
			OnStart:  start,
			OnStop:   stop,
			Position: positionAt(fset, call.Pos()),
		}, nil
	default:
		return model.Lifecycle{}, fmt.Errorf("unsupported lifecycle hook")
	}
}

func parseHookFunction(pkg *packages.Package, fset *token.FileSet, expr ast.Expr) (*model.Function, error) {
	if isNil(expr) {
		return nil, nil
	}

	fn, sig, err := parseFunction(pkg, expr)
	if err != nil {
		return nil, nodeMessage(fset, expr, err.Error())
	}

	if err := validateHookSignature(fn.Name(), sig); err != nil {
		return nil, nodeMessage(fset, expr, err.Error())
	}

	return &model.Function{
		Name:         fn.Name(),
		Position:     positionAt(fset, expr.Pos()),
		PackageName:  packageName(fn.Pkg()),
		PackagePath:  packagePath(fn.Pkg()),
		Inputs:       tupleTypes(sig.Params()),
		ReturnsError: sig.Results().Len() == 1,
	}, nil
}

func validateHookSignature(name string, sig *types.Signature) error {
	contextSeen := false
	for index := range sig.Params().Len() {
		param := sig.Params().At(index)
		if !isContextType(param.Type()) {
			continue
		}

		if index > 0 {
			return fmt.Errorf("hook %s context.Context must be the first parameter", name)
		}

		if contextSeen {
			return fmt.Errorf("hook %s can only accept one context.Context parameter", name)
		}

		contextSeen = true
	}

	switch sig.Results().Len() {
	case 0:
	case 1:
		if !isErrorType(sig.Results().At(0).Type()) {
			return fmt.Errorf("hook %s result must be error", name)
		}
	default:
		return fmt.Errorf("hook %s must return nothing or error", name)
	}

	return nil
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
	switch value := unwrap(expr).(type) {
	case *ast.Ident:
		return info.ObjectOf(value)
	case *ast.SelectorExpr:
		return info.ObjectOf(value.Sel)
	default:
		return nil
	}
}

func moduleObject(info *types.Info, expr ast.Expr) types.Object {
	switch value := unwrap(expr).(type) {
	case *ast.Ident:
		return info.ObjectOf(value)
	case *ast.SelectorExpr:
		return info.ObjectOf(value.Sel)
	default:
		return nil
	}
}

func bootstrapCall(info *types.Info, expr ast.Expr, names ...string) (string, bool) {
	selector, ok := unwrap(expr).(*ast.SelectorExpr)
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
	lit, ok := unwrap(expr).(*ast.BasicLit)
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

func isContextType(typ types.Type) bool {
	return typeKey(typ) == "context.Context"
}

func isBootstrapInType(typ types.Type) bool {
	return typeKey(typ) == bootstrapPackagePath+".In"
}

func structType(typ types.Type) (*types.Struct, bool) {
	switch value := typ.(type) {
	case *types.Named:
		strct, ok := value.Underlying().(*types.Struct)
		return strct, ok
	case *types.Struct:
		return value, true
	default:
		return nil, false
	}
}

func isNil(expr ast.Expr) bool {
	ident, ok := unwrap(expr).(*ast.Ident)
	return ok && ident.Name == "nil"
}

func unwrap(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
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

func typeKey(typ types.Type) string {
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}

		return pkg.Path()
	})
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

func collectPackages(root *packages.Package) map[string]*packages.Package {
	collected := map[string]*packages.Package{}

	var visit func(*packages.Package)
	visit = func(pkg *packages.Package) {
		if pkg == nil || pkg.PkgPath == "" {
			return
		}
		if _, exists := collected[pkg.PkgPath]; exists {
			return
		}

		collected[pkg.PkgPath] = pkg
		for _, imported := range pkg.Imports {
			visit(imported)
		}
	}

	visit(root)
	return collected
}

func (scan *scanner) packageIndex(pkg *packages.Package) *packageIndex {
	if pkg == nil {
		return &packageIndex{}
	}

	if index, ok := scan.indexes[pkg.PkgPath]; ok {
		return index
	}

	index := &packageIndex{
		values: map[types.Object]ast.Expr{},
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}

			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || len(valueSpec.Values) == 0 {
					continue
				}

				for indexName, name := range valueSpec.Names {
					obj := pkg.TypesInfo.Defs[name]
					if obj == nil {
						continue
					}

					value := valueFor(valueSpec.Values, indexName)
					if value != nil {
						index.values[obj] = value
					}
				}
			}
		}
	}

	scan.indexes[pkg.PkgPath] = index
	return index
}

func valueFor(values []ast.Expr, index int) ast.Expr {
	if len(values) == 1 {
		return values[0]
	}

	if index < len(values) {
		return values[index]
	}

	return nil
}

func moduleKey(obj types.Object) string {
	if obj == nil {
		return ""
	}

	return fmt.Sprintf("%s:%d:%s", packagePath(obj.Pkg()), obj.Pos(), obj.Name())
}
