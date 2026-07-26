package emit

import (
	"github.com/dave/jennifer/jen"
)

// generateConnectForwarder generates:
//
//	// ForwardToConnect creates a <ServiceName>MCPHandler that forwards
//	// MCP tool calls to a ConnectRPC client.
//	func <ServiceName>ForwardToConnect(client <connect_client_interface>) <ServiceName>MCPHandler
//
// This is opt-in and only generated when requested.
func generateConnectForwarder(f *jen.File, info ServiceEmitInfo) {
	funcName := info.Service.Name + "ForwardToConnect"
	handlerName := info.Service.Name + "MCPHandler"
	clientTypeName := info.ConnectClientType
	if clientTypeName == "" {
		clientTypeName = info.Service.Name + "ServiceClient"
	}
	
	// Assume Connect interface is in the same package as GoImportPath
	clientInterface := jen.Qual(info.GoImportPath, clientTypeName)
	
	structName := "connectForwarder" + info.Service.Name
	
	f.Type().Id(structName).Struct(
		jen.Id("client").Add(clientInterface),
	)
	
	f.Commentf("%s creates a %s that forwards MCP tool calls to a ConnectRPC client.", funcName, handlerName)
	f.Func().Id(funcName).Params(
		jen.Id("client").Add(clientInterface),
	).Id(handlerName).Block(
		jen.Return(jen.Op("&").Id(structName).Values(jen.Dict{
			jen.Id("client"): jen.Id("client"),
		})),
	)
	
	for _, t := range info.Tools {
		mcpRuntimePkg := "github.com/protocgen/proto2mcp/pkg/mcpruntime"

		f.Func().Params(
			jen.Id("f").Op("*").Id(structName),
		).Id(t.Tool.MethodName).Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("req").Op("*").Qual(t.InputType.ImportPath, t.InputType.TypeName),
		).Params(
			jen.Op("*").Qual(t.OutputType.ImportPath, t.OutputType.TypeName),
			jen.Error(),
		).Block(
			jen.Id("connectReq").Op(":=").Qual(connectPkg, "NewRequest").Call(jen.Id("req")),
			// Propagate headers from context (e.g., auth tokens, trace IDs).
			jen.If(
				jen.Id("headers").Op(":=").Qual(mcpRuntimePkg, "HeadersFromContext").Call(jen.Id("ctx")),
				jen.Id("headers").Op("!=").Nil(),
			).Block(
				jen.For(
					jen.List(jen.Id("k"), jen.Id("vals")).Op(":=").Range().Id("headers"),
				).Block(
					jen.For(
						jen.List(jen.Id("_"), jen.Id("v")).Op(":=").Range().Id("vals"),
					).Block(
						jen.Id("connectReq").Dot("Header").Call().Dot("Add").Call(jen.Id("k"), jen.Id("v")),
					),
				),
			),
			jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("f").Dot("client").Dot(t.Tool.MethodName).Call(
				jen.Id("ctx"),
				jen.Id("connectReq"),
			),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.Return(jen.Id("resp").Dot("Msg"), jen.Nil()),
		)
	}
}
