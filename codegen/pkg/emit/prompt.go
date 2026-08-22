package emit

import (
	"github.com/dave/jennifer/jen"
	"github.com/protocgen/proto2mcp/codegen/pkg/extract"
)

// PromptEmitInfo holds information needed to emit prompt code.
type PromptEmitInfo struct {
	Prompt    extract.PromptIR
	GoPackage string
}

// generatePromptInterface generates:
//
//	type <PromptName>PromptHandler interface {
//	    Handle<PromptName>(ctx context.Context, arguments map[string]string) (*mcpruntime.GetPromptResult, error)
//	}
func generatePromptInterface(f *jen.File, prompt extract.PromptIR) {
	interfaceName := prompt.Name + "PromptHandler"
	methodName := "Handle" + prompt.Name

	f.Commentf("%s is the interface for the %s prompt template.", interfaceName, prompt.Name)
	f.Type().Id(interfaceName).Interface(
		jen.Id(methodName).Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("arguments").Map(jen.String()).String(),
		).Params(
			jen.Op("*").Qual(runtimePkg, "GetPromptResult"),
			jen.Error(),
		),
	)
}

// generateRegisterPrompts generates:
//
//	func Register<File>Prompts(registry mcpruntime.PromptRegistrar, handler <PromptName>PromptHandler, ...) {
//	    registry.RegisterPrompt(mcpruntime.PromptDefinition{...}, handler.Handle<PromptName>)
//	}
func generateRegisterPrompts(f *jen.File, prompts []extract.PromptIR, funcName string) {
	if len(prompts) == 0 {
		return
	}

	// Build params: registry + one handler per prompt
	var params []jen.Code
	params = append(params, jen.Id("registry").Qual(runtimePkg, "PromptRegistrar"))
	for _, p := range prompts {
		params = append(params, jen.Id("handler"+p.Name).Id(p.Name+"PromptHandler"))
	}

	var body []jen.Code
	for _, p := range prompts {
		// Build arguments list
		var argValues []jen.Code
		for _, arg := range p.Arguments {
			argDict := jen.Dict{
				jen.Id("Name"): jen.Lit(arg.Name),
			}
			if arg.Description != "" {
				argDict[jen.Id("Description")] = jen.Lit(arg.Description)
			}
			if arg.Required {
				argDict[jen.Id("Required")] = jen.True()
			}
			argValues = append(argValues, jen.Qual(runtimePkg, "PromptArgument").Values(argDict))
		}

		defDict := jen.Dict{
			jen.Id("Name"): jen.Lit(p.Name),
		}
		if p.Description != "" {
			defDict[jen.Id("Description")] = jen.Lit(p.Description)
		}
		if len(argValues) > 0 {
			defDict[jen.Id("Arguments")] = jen.Index().Qual(runtimePkg, "PromptArgument").Values(argValues...)
		}

		body = append(body,
			jen.Id("registry").Dot("RegisterPrompt").Call(
				jen.Qual(runtimePkg, "PromptDefinition").Values(defDict),
				jen.Id("handler"+p.Name).Dot("Handle"+p.Name),
			),
		)
	}

	f.Commentf("%s registers all prompt templates.", funcName)
	f.Func().Id(funcName).Params(params...).Block(body...)
}

// GeneratePrompts produces prompt handler interfaces and registration.
func GeneratePrompts(f *jen.File, prompts []extract.PromptIR, registerFuncName string) {
	if len(prompts) == 0 {
		return
	}
	for _, p := range prompts {
		generatePromptInterface(f, p)
	}
	generateRegisterPrompts(f, prompts, registerFuncName)
}
