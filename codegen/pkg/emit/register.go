package emit

import (
	"github.com/dave/jennifer/jen"
)

// generateRegisterFunc generates:
//
//	func Register<ServiceName>MCP(registry *mcpruntime.ToolRegistry, handler <ServiceName>MCPHandler, opts ...mcpruntime.Option) {
//	    cfg := mcpruntime.NewConfig(opts...)
//	    // For each tool:
//	    registry.Register(mcpruntime.ToolDefinition{...}, cfg.WrapHandler("toolName", func(ctx context.Context, req mcpruntime.ToolRequest) (*mcpruntime.CallToolResult, error) {
//	        var input InputType
//	        if err := mcpruntime.UnmarshalToolInput(req, &input); err != nil {
//	            return mcpruntime.InvalidParamsError(err), nil
//	        }
//	        resp, err := handler.MethodName(ctx, &input)
//	        if err != nil {
//	            return mcpruntime.MapConnectError(err), nil
//	        }
//	        return mcpruntime.MarshalToolResult(resp)
//	    }))
//	}
func generateRegisterFunc(f *jen.File, info ServiceEmitInfo) {
	funcName := "Register" + info.Service.Name + "MCP"
	handlerName := info.Service.Name + "MCPHandler"

	f.Func().Id(funcName).Params(
		jen.Id("registry").Qual(runtimePkg, "Registry"),
		jen.Id("handler").Id(handlerName),
		jen.Id("opts").Op("...").Qual(runtimePkg, "Option"),
	).BlockFunc(func(g *jen.Group) {
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

	// Emit Annotations map with tool metadata hints.
	// Per MCP spec, destructiveHint defaults to true when omitted,
	// so we always emit both to make semantics explicit.
	annotations := []jen.Code{
		jen.Lit("readOnlyHint").Op(":").Lit(t.Tool.IsReadOnly),
		jen.Lit("destructiveHint").Op(":").Lit(t.Tool.IsDestructive),
	}
	defDict[jen.Id("Annotations")] = jen.Map(jen.String()).Any().Values(annotations...)

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

	g.Id("registry").Dot("Register").Call(
		def,
		jen.Id("cfg").Dot("WrapHandler").Call(jen.Id(nameConst), handlerClosure),
	)
}
