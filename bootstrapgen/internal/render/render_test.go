package render

import (
	"go/types"
	"testing"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/resolve"
)

func TestGoRendersBootstrapFunction(t *testing.T) {
	configPkg := types.NewPackage("example.com/app/config", "config")
	mainPkg := types.NewPackage("example.com/app/cmd/api", "main")
	configType := types.NewNamed(types.NewTypeName(0, configPkg, "Config", nil), types.NewStruct(nil, nil), nil)

	plan := &resolve.Plan{
		Spec: &model.Spec{
			PackageName: "main",
			PackagePath: "example.com/app/cmd/api",
			Entry: model.Entry{
				Name:         "run",
				PackageName:  "main",
				PackagePath:  "example.com/app/cmd/api",
				Inputs:       []types.Type{types.NewPointer(configType)},
				ReturnsError: true,
			},
		},
		Steps: []resolve.Step{
			{
				Provider: &model.Provider{
					Name:        "Load",
					PackageName: "config",
					PackagePath: "example.com/app/config",
					Output:      types.NewPointer(configType),
					HasError:    true,
				},
			},
		},
		Entry: []resolve.Source{
			{
				Kind: resolve.ProviderSource,
				Type: types.NewPointer(configType),
				Provider: &model.Provider{
					Name:        "Load",
					PackageName: "config",
					PackagePath: "example.com/app/config",
					Output:      types.NewPointer(configType),
					HasError:    true,
				},
			},
		},
	}

	_ = mainPkg

	code, err := Go(plan)
	if err != nil {
		t.Fatal(err)
	}

	expected := "package main\n\nimport (\n\t\"context\"\n\t\"example.com/app/config\"\n)\n\nfunc runBootstrap(ctx context.Context) error {\n\tconfig, err := config.Load()\n\tif err != nil {\n\t\treturn err\n\t}\n\treturn run(config)\n}\n"
	if string(code) != expected {
		t.Fatalf("unexpected generated code:\n%s", string(code))
	}
}
