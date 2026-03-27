package resolve

import (
	"go/types"
	"strings"
	"testing"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
)

func TestBuildMissingProviderErrorIncludesDependencyPath(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	configType := types.NewNamed(types.NewTypeName(0, pkg, "Config", nil), types.NewStruct(nil, nil), nil)
	serverType := types.NewNamed(types.NewTypeName(0, pkg, "Server", nil), types.NewStruct(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 20, Column: 1},
			Inputs: []model.EntryInput{
				{Type: types.NewPointer(serverType)},
			},
		},
		Providers: []model.Provider{
			{
				Name:     "NewServer",
				Position: model.Position{File: "bootstrap.go", Line: 10, Column: 1},
				Inputs:   []types.Type{types.NewPointer(configType)},
				Output:   types.NewPointer(serverType),
			},
		},
	}

	_, err := Build(spec)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	for _, fragment := range []string{
		"provider not found for *example.com/app.Config",
		"dependency path:",
		"entry run at bootstrap.go:20:1 requires *example.com/app.Server",
		"provider NewServer at bootstrap.go:10:1 requires *example.com/app.Config",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("missing fragment %q in error: %s", fragment, message)
		}
	}
}

func TestBuildMultipleProvidersErrorIncludesCandidates(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	serverType := types.NewNamed(types.NewTypeName(0, pkg, "Server", nil), types.NewStruct(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 30, Column: 1},
			Inputs: []model.EntryInput{
				{Type: types.NewPointer(serverType)},
			},
		},
		Providers: []model.Provider{
			{
				Name:     "NewServerA",
				Position: model.Position{File: "bootstrap.go", Line: 10, Column: 1},
				Output:   types.NewPointer(serverType),
			},
			{
				Name:     "NewServerB",
				Position: model.Position{File: "bootstrap.go", Line: 14, Column: 1},
				Output:   types.NewPointer(serverType),
			},
		},
	}

	_, err := Build(spec)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	for _, fragment := range []string{
		"multiple providers for *example.com/app.Server",
		"candidates:",
		"NewServerA at bootstrap.go:10:1 returns *example.com/app.Server",
		"NewServerB at bootstrap.go:14:1 returns *example.com/app.Server",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("missing fragment %q in error: %s", fragment, message)
		}
	}
}

func TestBuildResolvesEntryParamsFieldsAndHookDependencies(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	contextPkg := types.NewPackage("context", "context")
	configType := types.NewNamed(types.NewTypeName(0, pkg, "Config", nil), types.NewStruct(nil, nil), nil)
	serverType := types.NewNamed(types.NewTypeName(0, pkg, "Server", nil), types.NewStruct(nil, nil), nil)
	runnerType := types.NewNamed(types.NewTypeName(0, pkg, "Runner", nil), types.NewInterfaceType(nil, nil), nil)
	contextType := types.NewNamed(types.NewTypeName(0, contextPkg, "Context", nil), types.NewInterfaceType(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 30, Column: 1},
			Inputs: []model.EntryInput{
				{Type: contextType},
				{
					Type: types.NewStruct(nil, nil),
					Fields: []model.Field{
						{Name: "Runner", Type: runnerType, Position: model.Position{File: "bootstrap.go", Line: 31, Column: 2}},
					},
				},
			},
		},
		Bindings: []model.Binding{
			{
				Interface:      runnerType,
				Implementation: types.NewPointer(serverType),
				Position:       model.Position{File: "bootstrap.go", Line: 15, Column: 1},
			},
		},
		Providers: []model.Provider{
			{
				Name:     "NewConfig",
				Position: model.Position{File: "bootstrap.go", Line: 10, Column: 1},
				Output:   types.NewPointer(configType),
			},
			{
				Name:     "NewServer",
				Position: model.Position{File: "bootstrap.go", Line: 20, Column: 1},
				Inputs:   []types.Type{types.NewPointer(configType)},
				Output:   types.NewPointer(serverType),
			},
		},
		Lifecycles: []model.Lifecycle{
			{
				Kind: model.HookFuncLifecycle,
				OnStart: &model.Function{
					Name:     "startHook",
					Position: model.Position{File: "bootstrap.go", Line: 40, Column: 1},
					Inputs:   []types.Type{types.NewPointer(serverType)},
				},
			},
		},
	}

	plan, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.Entry) != 2 || len(plan.Entry[1].Fields) != 1 {
		t.Fatalf("unexpected entry plan: %+v", plan.Entry)
	}

	if plan.Entry[1].Fields[0].Source.Provider == nil || plan.Entry[1].Fields[0].Source.Provider.Name != "NewServer" {
		t.Fatalf("unexpected params field source: %+v", plan.Entry[1].Fields[0].Source)
	}

	if len(plan.Lifecycles) != 1 || plan.Lifecycles[0].Start == nil {
		t.Fatalf("unexpected lifecycle plan: %+v", plan.Lifecycles)
	}

	if len(plan.Lifecycles[0].Start.Inputs) != 1 || plan.Lifecycles[0].Start.Inputs[0].Provider.Name != "NewServer" {
		t.Fatalf("unexpected hook dependencies: %+v", plan.Lifecycles[0].Start)
	}
}

func TestBuildPrefersOverrideProvidersAndBindings(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	serverType := types.NewNamed(types.NewTypeName(0, pkg, "Server", nil), types.NewStruct(nil, nil), nil)
	runnerType := types.NewNamed(types.NewTypeName(0, pkg, "Runner", nil), types.NewInterfaceType(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 20, Column: 1},
			Inputs: []model.EntryInput{
				{Type: runnerType},
			},
		},
		Providers: []model.Provider{
			{
				Name:     "NewServer",
				Position: model.Position{File: "bootstrap.go", Line: 10, Column: 1},
				Output:   types.NewPointer(serverType),
			},
		},
		Overrides: []model.Provider{
			{
				Name:     "NewOverrideServer",
				Position: model.Position{File: "bootstrap.go", Line: 14, Column: 1},
				Output:   types.NewPointer(serverType),
			},
		},
		Bindings: []model.Binding{
			{
				Interface:      runnerType,
				Implementation: types.NewPointer(serverType),
				Position:       model.Position{File: "bootstrap.go", Line: 12, Column: 1},
			},
		},
	}

	plan, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.Entry) != 1 || plan.Entry[0].Source.Provider == nil {
		t.Fatalf("unexpected entry plan: %+v", plan.Entry)
	}

	if plan.Entry[0].Source.Provider.Name != "NewOverrideServer" {
		t.Fatalf("override provider was not selected: %+v", plan.Entry[0].Source.Provider)
	}
}

func TestBuildRejectsDuplicateOverrideBindings(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	runnerType := types.NewNamed(types.NewTypeName(0, pkg, "Runner", nil), types.NewInterfaceType(nil, nil), nil)
	serverAType := types.NewNamed(types.NewTypeName(0, pkg, "ServerA", nil), types.NewStruct(nil, nil), nil)
	serverBType := types.NewNamed(types.NewTypeName(0, pkg, "ServerB", nil), types.NewStruct(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 20, Column: 1},
		},
		OverrideBindings: []model.Binding{
			{
				Interface:      runnerType,
				Implementation: types.NewPointer(serverAType),
				Position:       model.Position{File: "bootstrap.go", Line: 10, Column: 1},
			},
			{
				Interface:      runnerType,
				Implementation: types.NewPointer(serverBType),
				Position:       model.Position{File: "bootstrap.go", Line: 11, Column: 1},
			},
		},
	}

	_, err := Build(spec)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "duplicate binding for example.com/app.Runner") {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestBuildMissingProviderErrorIncludesEntryFieldPosition(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	serviceType := types.NewNamed(types.NewTypeName(0, pkg, "Service", nil), types.NewStruct(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 30, Column: 1},
			Inputs: []model.EntryInput{
				{
					Type: types.NewStruct(nil, nil),
					Fields: []model.Field{
						{Name: "Service", Type: types.NewPointer(serviceType), Position: model.Position{File: "bootstrap.go", Line: 31, Column: 2}},
					},
				},
			},
		},
	}

	_, err := Build(spec)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	for _, fragment := range []string{
		"provider not found for *example.com/app.Service",
		"entry field run.Service at bootstrap.go:31:2 requires *example.com/app.Service",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("missing fragment %q in error: %s", fragment, message)
		}
	}
}

func TestBuildMissingProviderErrorIncludesLifecycleHookDependencyPath(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	configType := types.NewNamed(types.NewTypeName(0, pkg, "Config", nil), types.NewStruct(nil, nil), nil)
	serverType := types.NewNamed(types.NewTypeName(0, pkg, "Server", nil), types.NewStruct(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 50, Column: 1},
		},
		Providers: []model.Provider{
			{
				Name:     "NewServer",
				Position: model.Position{File: "bootstrap.go", Line: 20, Column: 1},
				Inputs:   []types.Type{types.NewPointer(configType)},
				Output:   types.NewPointer(serverType),
			},
		},
		Lifecycles: []model.Lifecycle{
			{
				Kind: model.HookFuncLifecycle,
				OnStart: &model.Function{
					Name:     "startHook",
					Position: model.Position{File: "bootstrap.go", Line: 40, Column: 1},
					Inputs:   []types.Type{types.NewPointer(serverType)},
				},
			},
		},
	}

	_, err := Build(spec)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	for _, fragment := range []string{
		"provider not found for *example.com/app.Config",
		"lifecycle hook startHook at bootstrap.go:40:1 requires *example.com/app.Server",
		"provider NewServer at bootstrap.go:20:1 requires *example.com/app.Config",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("missing fragment %q in error: %s", fragment, message)
		}
	}
}

func TestBuildRejectsDuplicateOverrideProviders(t *testing.T) {
	pkg := types.NewPackage("example.com/app", "app")
	serverType := types.NewNamed(types.NewTypeName(0, pkg, "Server", nil), types.NewStruct(nil, nil), nil)

	spec := &model.Spec{
		Entry: model.Entry{
			Name:     "run",
			Position: model.Position{File: "bootstrap.go", Line: 20, Column: 1},
			Inputs: []model.EntryInput{
				{Type: types.NewPointer(serverType)},
			},
		},
		Overrides: []model.Provider{
			{
				Name:     "NewServerA",
				Position: model.Position{File: "bootstrap.go", Line: 10, Column: 1},
				Output:   types.NewPointer(serverType),
			},
			{
				Name:     "NewServerB",
				Position: model.Position{File: "bootstrap.go", Line: 12, Column: 1},
				Output:   types.NewPointer(serverType),
			},
		},
	}

	_, err := Build(spec)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	for _, fragment := range []string{
		"multiple providers for *example.com/app.Server",
		"NewServerA at bootstrap.go:10:1 returns *example.com/app.Server",
		"NewServerB at bootstrap.go:12:1 returns *example.com/app.Server",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("missing fragment %q in error: %s", fragment, message)
		}
	}
}
