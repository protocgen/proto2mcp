package emit

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/protocgen/proto2mcp/codegen/pkg/extract"
)

func TestE2E_FullPipeline(t *testing.T) {
	serviceInfo := ServiceEmitInfo{
		GoPackage:    "example",
		GoImportPath: "github.com/example/example",
		Service: extract.ServiceIR{
			Name:        "TodoService",
			FullName:    "example.v1.TodoService",
			Description: "Manages todos",
		},
		Tools: []ToolEmitInfo{
			{
				Tool: extract.ToolIR{
					Name:        "TodoService_CreateTodo",
					MethodName:  "CreateTodo",
					Description: "Create a new todo item",
					InputSchema: []byte(`{"type":"object","properties":{"title":{"type":"string"}}}`),
				},
				InputType: TypeRef{
					ImportPath: "github.com/example/example/v1",
					TypeName:   "CreateTodoRequest",
				},
				OutputType: TypeRef{
					ImportPath: "github.com/example/example/v1",
					TypeName:   "CreateTodoResponse",
				},
			},
			{
				Tool: extract.ToolIR{
					Name:         "TodoService_GetTodo",
					MethodName:   "GetTodo",
					Description:  "Get a todo by ID",
					InputSchema:  []byte(`{"type":"object","properties":{"id":{"type":"string"}}}`),
					ResourceKeys: []string{"id"},
					IsReadOnly:   true,
				},
				InputType: TypeRef{
					ImportPath: "github.com/example/example/v1",
					TypeName:   "GetTodoRequest",
				},
				OutputType: TypeRef{
					ImportPath: "github.com/example/example/v1",
					TypeName:   "GetTodoResponse",
				},
			},
		},
	}

	prompts := []extract.PromptIR{
		{
			Name:        "OnboardUser",
			Description: "Guide user onboarding",
			Arguments: []extract.PromptArgIR{
				{
					Name:        "username",
					Description: "The username to onboard",
					Required:    true,
				},
			},
		},
	}

	f := GenerateFile([]ServiceEmitInfo{serviceInfo})
	GeneratePrompts(f, prompts, "RegisterTodoPromptsMCP")

	var buf bytes.Buffer
	if err := f.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	code := buf.String()

	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "generated.go", code, parser.AllErrors); err != nil {
		t.Errorf("generated code does not parse: %v\n%s", err, code)
	}

	wants := []string{
		"TodoServiceMCPHandler",
		"RegisterTodoServiceMCP",
		"TodoService_CreateTodoName",
		"TodoService_GetTodoName",
		"OnboardUserPromptHandler",
		"RegisterTodoPromptsMCP",
		"HandleOnboardUser",
	}

	for _, want := range wants {
		if !strings.Contains(code, want) {
			t.Errorf("generated code missing symbol %q", want)
		}
	}
}
