package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/protocgen/proto2mcp/codegen/pkg/emit"
	"github.com/protocgen/proto2mcp/codegen/pkg/extract"
)

func main() {
	log.SetPrefix("protoc-gen-proto2mcp: ")
	log.SetFlags(0)

	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		// Extract IR from all files.
		fileIRs, err := extract.FromPlugin(gen)
		if err != nil {
			return err
		}

		// Build a map from proto file path to protogen.File for type resolution.
		fileMap := make(map[string]*protogen.File)
		for _, f := range gen.Files {
			fileMap[f.Desc.Path()] = f
		}

		for _, fileIR := range fileIRs {
			if fileIR.Skip {
				continue
			}

			// Print warnings to stderr.
			for _, w := range fileIR.Warnings {
				fmt.Fprintf(os.Stderr, "%s: %s [%s]\n", w.Method, w.Message, severityStr(w.Severity))
			}

			protoFile := fileMap[fileIR.FileName]
			if protoFile == nil {
				continue
			}

			var infos []emit.ServiceEmitInfo
			for _, svcIR := range fileIR.Services {
				// Find the matching protogen.Service for type resolution.
				var protoSvc *protogen.Service
				for _, s := range protoFile.Services {
					if string(s.Desc.FullName()) == svcIR.FullName {
						protoSvc = s
						break
					}
				}
				if protoSvc == nil {
					continue
				}

				// Build emit info with resolved Go types.
				info := buildServiceEmitInfo(svcIR, protoSvc, protoFile)
				infos = append(infos, info)
			}

			if len(infos) == 0 {
				continue
			}

			// Generate the code.
			jFile := emit.GenerateFile(infos)

			// Determine output filename: input.pb.mcp.go
			outputName := strings.TrimSuffix(protoFile.GeneratedFilenamePrefix, ".pb") + ".pb.mcp.go"
			// If the prefix doesn't end with .pb, just append.
			if !strings.Contains(protoFile.GeneratedFilenamePrefix, ".pb") {
				outputName = protoFile.GeneratedFilenamePrefix + "_mcp.pb.go"
			}

			g := gen.NewGeneratedFile(outputName, protoFile.GoImportPath)
			if err := jFile.Render(g); err != nil {
				return fmt.Errorf("writing %s: %w", outputName, err)
			}
		}
		return nil
	})
}

// buildServiceEmitInfo resolves Go type references from protogen metadata.
func buildServiceEmitInfo(svcIR extract.ServiceIR, protoSvc *protogen.Service, protoFile *protogen.File) emit.ServiceEmitInfo {
	info := emit.ServiceEmitInfo{
		Service:      svcIR,
		GoPackage:    string(protoFile.GoPackageName),
		GoImportPath: string(protoFile.GoImportPath),
	}

	// Build a map from method name to protogen.Method for type resolution.
	methodMap := make(map[string]*protogen.Method)
	for _, m := range protoSvc.Methods {
		methodMap[string(m.Desc.Name())] = m
	}

	for _, toolIR := range svcIR.Tools {
		protoMethod := methodMap[toolIR.MethodName]
		if protoMethod == nil {
			continue
		}

		toolInfo := emit.ToolEmitInfo{
			Tool: toolIR,
			InputType: emit.TypeRef{
				ImportPath: string(protoMethod.Input.GoIdent.GoImportPath),
				TypeName:   protoMethod.Input.GoIdent.GoName,
			},
			OutputType: emit.TypeRef{
				ImportPath: string(protoMethod.Output.GoIdent.GoImportPath),
				TypeName:   protoMethod.Output.GoIdent.GoName,
			},
		}
		info.Tools = append(info.Tools, toolInfo)
	}

	return info
}

func severityStr(s extract.WarningLevel) string {
	switch s {
	case extract.WarnInfo:
		return "INFO"
	case extract.WarnWarning:
		return "WARNING"
	case extract.WarnError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
