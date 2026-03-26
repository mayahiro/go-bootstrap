package render

import (
	"go/types"
	"strings"
	"testing"

	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/model"
	"github.com/mayahiro/go-bootstrap/bootstrapgen/internal/resolve"
)

func TestGoRendersBootstrapFunction(t *testing.T) {
	configPkg := types.NewPackage("example.com/app/config", "config")
	servicePkg := types.NewPackage("example.com/app/internal/service", "service")
	youtubePkg := types.NewPackage("example.com/app/internal/youtube", "youtube")
	mainPkg := types.NewPackage("example.com/app/cmd/api", "main")

	configType := types.NewNamed(types.NewTypeName(0, configPkg, "Config", nil), types.NewStruct(nil, nil), nil)
	serviceType := types.NewNamed(types.NewTypeName(0, servicePkg, "Service", nil), types.NewStruct(nil, nil), nil)
	youtubeServiceType := types.NewNamed(types.NewTypeName(0, youtubePkg, "Service", nil), types.NewStruct(nil, nil), nil)
	paramsType := types.NewNamed(types.NewTypeName(0, mainPkg, "Params", nil), types.NewStruct(nil, nil), nil)

	configProvider := &model.Provider{
		Name:        "Load",
		PackageName: "config",
		PackagePath: "example.com/app/config",
		Output:      types.NewPointer(configType),
		HasError:    true,
	}
	serviceProvider := &model.Provider{
		Name:        "New",
		PackageName: "service",
		PackagePath: "example.com/app/internal/service",
		Output:      types.NewPointer(serviceType),
	}
	youtubeProvider := &model.Provider{
		Name:        "New",
		PackageName: "youtube",
		PackagePath: "example.com/app/internal/youtube",
		Output:      types.NewPointer(youtubeServiceType),
	}

	plan := &resolve.Plan{
		Spec: &model.Spec{
			PackageName: "main",
			PackagePath: "example.com/app/cmd/api",
			Entry: model.Entry{
				Name:         "run",
				PackageName:  "main",
				PackagePath:  "example.com/app/cmd/api",
				Inputs: []model.EntryInput{
					{Type: types.NewPointer(configType)},
					{
						Type: paramsType,
						Fields: []model.Field{
							{Name: "Service", Type: types.NewPointer(youtubeServiceType)},
						},
					},
				},
				ReturnsError: true,
			},
		},
		Steps: []resolve.Step{
			{Provider: configProvider},
			{Provider: serviceProvider},
			{Provider: youtubeProvider},
		},
		Entry: []resolve.EntryArg{
			{
				Type: types.NewPointer(configType),
				Source: resolve.Source{
					Kind:     resolve.ProviderSource,
					Type:     types.NewPointer(configType),
					Provider: configProvider,
				},
			},
			{
				Type: paramsType,
				Fields: []resolve.EntryField{
					{
						Name: "Service",
						Source: resolve.Source{
							Kind:     resolve.ProviderSource,
							Type:     types.NewPointer(youtubeServiceType),
							Provider: youtubeProvider,
						},
					},
				},
			},
		},
		Lifecycles: []resolve.Lifecycle{
			{
				Spec: model.Lifecycle{
					Kind: model.HookFuncLifecycle,
				},
				Start: &resolve.HookCall{
					Func: &model.Function{
						Name:         "StartAudit",
						PackageName:  "service",
						PackagePath:  "example.com/app/internal/service",
						Inputs:       []types.Type{types.NewPointer(serviceType)},
						ReturnsError: true,
					},
					Inputs: []resolve.Source{
						{
							Kind:     resolve.ProviderSource,
							Type:     types.NewPointer(serviceType),
							Provider: serviceProvider,
						},
					},
				},
			},
		},
	}

	code, err := Go(plan)
	if err != nil {
		t.Fatal(err)
	}

	generated := string(code)
	for _, fragment := range []string{
		"config2, err := config.Load()",
		"service2 := service.New()",
		"youtubeService := youtube.New()",
		"params := Params{",
		"Service: youtubeService,",
		"if err := service.StartAudit(service2); err != nil {",
		"return run(config2, params)",
	} {
		if !strings.Contains(generated, fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, generated)
		}
	}
}
