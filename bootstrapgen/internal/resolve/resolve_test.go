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
			Inputs:   []types.Type{types.NewPointer(serverType)},
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
			Inputs:   []types.Type{types.NewPointer(serverType)},
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
