package emit

import (
	"github.com/dave/jennifer/jen"
)

// generateRegisterFunc generates:
//
//	func Register<ServiceName>MCP(registry mcpruntime.Registry, handler <ServiceName>MCPHandler, opts ...mcpruntime.Option) {
//	    // Auto-configure WithToolRegistry when using *ToolRegistry.
//	    if tr, ok := registry.(*mcpruntime.ToolRegistry); ok {
//	        opts = append([]mcpruntime.Option{mcpruntime.WithToolRegistry(tr)}, opts...)
//	    }
//	    cfg := mcpruntime.NewConfig(opts...)
//	    // For each tool:
//	    registry.Register(mcpruntime.ToolDefinition{...}, cfg.WrapHandler("toolName", handlerClosure))
//	}
func generateRegisterFunc(f *jen.File, info ServiceEmitInfo) {
	funcName := "Register" + info.Service.Name + "MCP"
	handlerName := info.Service.Name + "MCPHandler"

	f.Func().Id(funcName).Params(
		jen.Id("registry").Qual(runtimePkg, "Registry"),
		jen.Id("handler").Id(handlerName),
		jen.Id("opts").Op("...").Qual(runtimePkg, "Option"),
	).BlockFunc(func(g *jen.Group) {
		// Auto-detect *ToolRegistry for ToolDefinition injection.
		g.If(
			jen.List(jen.Id("tr"), jen.Id("ok")).Op(":=").Id("registry").Assert(jen.Op("*").Qual(runtimePkg, "ToolRegistry")),
			jen.Id("ok"),
		).Block(
			jen.Id("opts").Op("=").Append(
				jen.Index().Qual(runtimePkg, "Option").Values(
					jen.Qual(runtimePkg, "WithToolRegistry").Call(jen.Id("tr")),
				),
				jen.Id("opts").Op("..."),
			),
		)

		if len(info.Tools) > 0 {
			g.Id("cfg").Op(":=").Qual(runtimePkg, "NewConfig").Call(jen.Id("opts").Op("..."))
			g.Line()
		}
		for _, t := range info.Tools {
			generateRegisterCall(g, t)
		}
	})
}

func generateRegisterCall(g *jen.Group, t ToolEmitInfo) {
	nameConst := t.Tool.Name + "Name"
	descConst := t.Tool.Name + "Description"
	schemaConst := t.Tool.Name + "Schema"

	defDict := jen.Dict{
		jen.Id("Name"):        jen.Id(nameConst),
		jen.Id("Description"): jen.Id(descConst),
		jen.Id("InputSchema"): jen.Qual("encoding/json", "RawMessage").Call(jen.Id(schemaConst)),
	}

	if t.Tool.IsReadOnly || t.Tool.IsDestructive {
		annDict := jen.Dict{}
		if t.Tool.IsReadOnly {
			annDict[jen.Lit("readOnlyHint")] = jen.True()
		}
		if t.Tool.IsDestructive {
			annDict[jen.Lit("destructiveHint")] = jen.True()
		}
		defDict[jen.Id("Annotations")] = jen.Map(jen.String()).Any().Values(annDict)
	}

	// Emit ResourceKeys if any fields are annotated with resource_key.
	if len(t.Tool.ResourceKeys) > 0 {
		keys := make([]jen.Code, len(t.Tool.ResourceKeys))
		for i, k := range t.Tool.ResourceKeys {
			keys[i] = jen.Lit(k)
		}
		defDict[jen.Id("ResourceKeys")] = jen.Index().String().Values(keys...)
	}

	if t.Tool.ResourceURI != "" {
		defDict[jen.Id("ResourceURI")] = jen.Lit(t.Tool.ResourceURI)
	}

	if len(t.Tool.SubTools) > 0 {
		var stepValues []jen.Code
		for _, step := range t.Tool.SubTools {
			stepDict := jen.Dict{
				jen.Id("ToolName"): jen.Lit(step.ToolName),
			}
			if step.OutputKey != "" {
				stepDict[jen.Id("OutputKey")] = jen.Lit(step.OutputKey)
			}
			stepValues = append(stepValues, jen.Qual(runtimePkg, "MacroStep").Values(stepDict))
		}
		defDict[jen.Id("Steps")] = jen.Index().Qual(runtimePkg, "MacroStep").Values(stepValues...)
	}

	def := jen.Qual(runtimePkg, "ToolDefinition").Values(defDict)

	handlerClosure := jen.Func().Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("req").Qual(runtimePkg, "ToolRequest"),
	).Params(
		jen.Op("*").Qual(runtimePkg, "CallToolResult"),
		jen.Error(),
	).Block(
		jen.Var().Id("input").Qual(t.InputType.ImportPath, t.InputType.TypeName),
		jen.If(
			jen.Err().Op(":=").Qual(runtimePkg, "UnmarshalToolInput").Call(jen.Id("req"), jen.Op("&").Id("input")),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Qual(runtimePkg, "InvalidParamsError").Call(jen.Err()), jen.Nil()),
		),
		jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("handler").Dot(t.Tool.MethodName).Call(jen.Id("ctx"), jen.Op("&").Id("input")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Qual(connectBridgePkg, "MapConnectError").Call(jen.Err()), jen.Nil()),
		),
		jen.Return(jen.Qual(runtimePkg, "MarshalToolResult").Call(jen.Id("resp"))),
	)

	registerMethod := "Register"
	if len(t.Tool.SubTools) > 0 {
		registerMethod = "RegisterMacro"
	}

	g.Id("registry").Dot(registerMethod).Call(
		def,
		jen.Id("cfg").Dot("WrapHandler").Call(jen.Id(nameConst), handlerClosure),
	)
}
