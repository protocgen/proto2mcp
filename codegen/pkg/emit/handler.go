package emit

import (
	"github.com/dave/jennifer/jen"
)

// generateHandlerInterface generates:
//
//	type <ServiceName>MCPHandler interface {
//	    <MethodName>(ctx context.Context, req *<InputType>) (*<OutputType>, error)
//	}
func generateHandlerInterface(f *jen.File, info ServiceEmitInfo) {
	interfaceName := info.Service.Name + "MCPHandler"

	methods := []jen.Code{}
	for _, t := range info.Tools {
		method := jen.Id(t.Tool.MethodName).Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("req").Op("*").Qual(t.InputType.ImportPath, t.InputType.TypeName),
		).Params(
			jen.Op("*").Qual(t.OutputType.ImportPath, t.OutputType.TypeName),
			jen.Error(),
		)
		methods = append(methods, method)
	}

	f.Type().Id(interfaceName).Interface(methods...)
}
