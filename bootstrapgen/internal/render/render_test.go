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
				Name:        "run",
				PackageName: "main",
				PackagePath: "example.com/app/cmd/api",
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

func TestGoRendersStartStopAndCloseLifecycles(t *testing.T) {
	configPkg := types.NewPackage("example.com/app/config", "config")
	serverPkg := types.NewPackage("example.com/app/internal/httpserver", "httpserver")
	auditPkg := types.NewPackage("example.com/app/internal/audit", "audit")
	mainPkg := types.NewPackage("example.com/app/cmd/api", "main")
	contextPkg := types.NewPackage("context", "context")

	configType := types.NewNamed(types.NewTypeName(0, configPkg, "Config", nil), types.NewStruct(nil, nil), nil)
	serverType := types.NewNamed(types.NewTypeName(0, serverPkg, "Server", nil), types.NewStruct(nil, nil), nil)
	writerType := types.NewNamed(types.NewTypeName(0, auditPkg, "Writer", nil), types.NewStruct(nil, nil), nil)
	contextType := types.NewNamed(types.NewTypeName(0, contextPkg, "Context", nil), types.NewInterfaceType(nil, nil), nil)
	writerType.AddMethod(types.NewFunc(
		0,
		auditPkg,
		"Close",
		types.NewSignatureType(
			types.NewVar(0, auditPkg, "", types.NewPointer(writerType)),
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(types.NewVar(0, nil, "", types.Universe.Lookup("error").Type())),
			false,
		),
	))

	configProvider := &model.Provider{
		Name:        "Load",
		PackageName: "config",
		PackagePath: "example.com/app/config",
		Output:      types.NewPointer(configType),
		HasError:    true,
	}
	serverProvider := &model.Provider{
		Name:        "New",
		PackageName: "httpserver",
		PackagePath: "example.com/app/internal/httpserver",
		Inputs:      []types.Type{types.NewPointer(configType)},
		Output:      types.NewPointer(serverType),
	}
	writerProvider := &model.Provider{
		Name:        "NewWriter",
		PackageName: "audit",
		PackagePath: "example.com/app/internal/audit",
		Output:      types.NewPointer(writerType),
	}

	plan := &resolve.Plan{
		Spec: &model.Spec{
			PackageName: "main",
			PackagePath: "example.com/app/cmd/api",
			Entry: model.Entry{
				Name:         "run",
				PackageName:  packageName(mainPkg),
				PackagePath:  packagePath(mainPkg),
				Inputs:       []model.EntryInput{{Type: types.NewPointer(serverType)}},
				ReturnsError: true,
			},
		},
		Steps: []resolve.Step{
			{Provider: configProvider},
			{Provider: serverProvider, Inputs: []resolve.Source{{Kind: resolve.ProviderSource, Type: types.NewPointer(configType), Provider: configProvider}}},
			{Provider: writerProvider},
		},
		Entry: []resolve.EntryArg{
			{
				Type: types.NewPointer(serverType),
				Source: resolve.Source{
					Kind:     resolve.ProviderSource,
					Type:     types.NewPointer(serverType),
					Provider: serverProvider,
				},
			},
		},
		Lifecycles: []resolve.Lifecycle{
			{
				Spec: model.Lifecycle{
					Kind:   model.StartStopLifecycle,
					Target: types.NewPointer(serverType),
					OnStart: &model.Function{
						Name:         "Start",
						PackageName:  "httpserver",
						PackagePath:  "example.com/app/internal/httpserver",
						Inputs:       []types.Type{types.NewPointer(serverType)},
						ReturnsError: true,
					},
					OnStop: &model.Function{
						Name:         "Stop",
						PackageName:  "httpserver",
						PackagePath:  "example.com/app/internal/httpserver",
						Inputs:       []types.Type{types.NewPointer(serverType), contextType},
						ReturnsError: true,
					},
				},
				Source: resolve.Source{
					Kind:     resolve.ProviderSource,
					Type:     types.NewPointer(serverType),
					Provider: serverProvider,
				},
			},
			{
				Spec: model.Lifecycle{
					Kind:   model.CloseLifecycle,
					Target: types.NewPointer(writerType),
				},
				Source: resolve.Source{
					Kind:     resolve.ProviderSource,
					Type:     types.NewPointer(writerType),
					Provider: writerProvider,
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
		"httpserverServer := httpserver.New(config2)",
		"auditWriter := audit.NewWriter()",
		"if err := httpserverServer.Start(); err != nil {",
		"defer func() { _ = httpserverServer.Stop(ctx) }()",
		"defer func() { _ = auditWriter.Close() }()",
		"return run(httpserverServer)",
	} {
		if !strings.Contains(generated, fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, generated)
		}
	}
}

func TestGoRendersMultipleEntryParamStructs(t *testing.T) {
	mainPkg := types.NewPackage("example.com/app/cmd/api", "main")
	serverPkg := types.NewPackage("example.com/app/internal/server", "server")
	configPkg := types.NewPackage("example.com/app/internal/config", "config")
	contextPkg := types.NewPackage("context", "context")

	serverType := types.NewNamed(types.NewTypeName(0, serverPkg, "Server", nil), types.NewStruct(nil, nil), nil)
	configType := types.NewNamed(types.NewTypeName(0, configPkg, "Config", nil), types.NewStruct(nil, nil), nil)
	serverParamsType := types.NewNamed(types.NewTypeName(0, mainPkg, "ServerParams", nil), types.NewStruct(nil, nil), nil)
	configParamsType := types.NewNamed(types.NewTypeName(0, mainPkg, "ConfigParams", nil), types.NewStruct(nil, nil), nil)
	contextType := types.NewNamed(types.NewTypeName(0, contextPkg, "Context", nil), types.NewInterfaceType(nil, nil), nil)

	serverProvider := &model.Provider{
		Name:        "New",
		PackageName: "server",
		PackagePath: "example.com/app/internal/server",
		Output:      types.NewPointer(serverType),
	}
	configProvider := &model.Provider{
		Name:        "Load",
		PackageName: "config",
		PackagePath: "example.com/app/internal/config",
		Output:      types.NewPointer(configType),
	}

	plan := &resolve.Plan{
		Spec: &model.Spec{
			PackageName: "main",
			PackagePath: "example.com/app/cmd/api",
			Entry: model.Entry{
				Name:        "run",
				PackageName: packageName(mainPkg),
				PackagePath: packagePath(mainPkg),
				Inputs: []model.EntryInput{
					{Type: contextType},
					{
						Type: serverParamsType,
						Fields: []model.Field{
							{Name: "Server", Type: types.NewPointer(serverType)},
						},
					},
					{
						Type: configParamsType,
						Fields: []model.Field{
							{Name: "Config", Type: types.NewPointer(configType)},
						},
					},
				},
				ReturnsError: true,
			},
		},
		Steps: []resolve.Step{
			{Provider: serverProvider},
			{Provider: configProvider},
		},
		Entry: []resolve.EntryArg{
			{
				Type:   contextType,
				Source: resolve.Source{Kind: resolve.ContextSource},
			},
			{
				Type: serverParamsType,
				Fields: []resolve.EntryField{
					{
						Name: "Server",
						Source: resolve.Source{
							Kind:     resolve.ProviderSource,
							Type:     types.NewPointer(serverType),
							Provider: serverProvider,
						},
					},
				},
			},
			{
				Type: configParamsType,
				Fields: []resolve.EntryField{
					{
						Name: "Config",
						Source: resolve.Source{
							Kind:     resolve.ProviderSource,
							Type:     types.NewPointer(configType),
							Provider: configProvider,
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
		"server2 := server.New()",
		"config2 := config.Load()",
		"serverParams := ServerParams{",
		"Server: server2,",
		"configParams := ConfigParams{",
		"Config: config2,",
		"return run(ctx, serverParams, configParams)",
	} {
		if !strings.Contains(generated, fragment) {
			t.Fatalf("generated code did not include %q:\n%s", fragment, generated)
		}
	}
}
