package extract

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Extension field numbers for MCP annotations.
// Source: proto/protocgen/mcp/v1/options.proto
const (
	// extMethodOptions is the field number for MethodMCPOptions on google.protobuf.MethodOptions.
	extMethodOptions protoreflect.FieldNumber = 1179
	// extServiceOptions is the field number for ServiceMCPOptions on google.protobuf.ServiceOptions.
	extServiceOptions protoreflect.FieldNumber = 1180
	// extFileOptions is the field number for FileMCPOptions on google.protobuf.FileOptions.
	extFileOptions protoreflect.FieldNumber = 1181
)

// ExtractFile processes a single protogen.File into a FileIR.
// It walks all services and methods, reads MCP annotations,
// and produces the intermediate representation.
func ExtractFile(file *protogen.File) (*FileIR, error) {
	fOpts := readFileOptions(file)
	if fOpts != nil && fOpts.Skip {
		return &FileIR{
			FileName: file.Desc.Path(),
			Skip:     true,
		}, nil
	}

	ir := &FileIR{
		FileName: file.Desc.Path(),
		Skip:     false,
	}

	for _, svc := range file.Services {
		svcOpts := readServiceOptions(svc)
		prefix := ""
		if svcOpts != nil && svcOpts.ToolNamePrefix != "" {
			prefix = svcOpts.ToolNamePrefix
		}

		svcIR := extractService(svc, prefix)

		// If description is empty or overridden by options
		if svcOpts != nil && svcOpts.Description != "" {
			svcIR.Description = svcOpts.Description
		}

		if svcOpts != nil {
			svcIR.Options = ServiceOptions{
				ToolNamePrefix: svcOpts.ToolNamePrefix,
				Description:    svcOpts.Description,
			}
		}

		// Aggregate warnings from methods
		for _, tool := range svcIR.Tools {
			ir.Warnings = append(ir.Warnings, tool.Warnings...)
		}

		ir.Services = append(ir.Services, *svcIR)
	}

	// Fail the build if any WarnError-level warnings were found.
	for _, w := range ir.Warnings {
		if w.Severity == WarnError {
			return ir, fmt.Errorf("extraction error in %s: %s: %s", ir.FileName, w.Method, w.Message)
		}
	}

	return ir, nil
}

func extractService(svc *protogen.Service, svcPrefix string) *ServiceIR {
	svcIR := &ServiceIR{
		Name:        svc.GoName,
		FullName:    string(svc.Desc.FullName()),
		Description: strings.TrimSpace(svc.Comments.Leading.String()),
	}

	for _, method := range svc.Methods {
		toolIR := extractMethod(method, svc.GoName, svcPrefix)
		if toolIR != nil {
			svcIR.Tools = append(svcIR.Tools, *toolIR)
		}
	}

	return svcIR
}

func extractMethod(method *protogen.Method, svcName, svcPrefix string) *ToolIR {
	mOpts := readMethodOptions(method)
	if mOpts != nil && mOpts.Skip {
		return nil
	}

	toolName := ""
	if mOpts != nil && mOpts.ToolName != "" {
		toolName = mOpts.ToolName
	} else {
		if svcPrefix != "" {
			toolName = svcPrefix + "_" + method.GoName
		} else {
			toolName = svcName + "_" + method.GoName
		}
	}

	desc := strings.TrimSpace(method.Comments.Leading.String())
	if mOpts != nil && mOpts.Description != "" {
		desc = mOpts.Description
	}

	isDeprecated := false
	if method.Desc.Options() != nil {
		if mOptsPb, ok := method.Desc.Options().(*descriptorpb.MethodOptions); ok && mOptsPb.GetDeprecated() {
			isDeprecated = true
		}
	}
	if mOpts != nil && mOpts.IsDeprecated {
		isDeprecated = true
	}

	if isDeprecated {
		desc = "[DEPRECATED] " + desc
	}

	tool := &ToolIR{
		Name:           toolName,
		MethodName:     string(method.Desc.Name()),
		Description:    desc,
		InputTypeName:  string(method.Input.Desc.FullName()),
		OutputTypeName: string(method.Output.Desc.FullName()),
		IsDeprecated:   isDeprecated,
	}

	if mOpts != nil {
		tool.IsResource = mOpts.Resource
		tool.ResourceURI = mOpts.ResourceURITemplate
		tool.IsReadOnly = mOpts.ReadOnly
		tool.IsDestructive = mOpts.Destructive
		tool.Version = mOpts.Version
	}

	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		tool.Warnings = append(tool.Warnings, Warning{
			Severity: WarnError,
			Method:   toolName,
			Message:  "streaming methods are not supported",
		})
	}

	// InputSchema will be wired up later
	tool.InputSchema = nil

	return tool
}

type methodOptions struct {
	ToolName            string
	Description         string
	Skip                bool
	Resource            bool
	ResourceURITemplate string
	ReadOnly            bool
	Destructive         bool
	Version             int32
	IsDeprecated        bool
}

type serviceOptions struct {
	ToolNamePrefix string
	Description    string
}

type fileOptions struct {
	Skip bool
}

func readMethodOptions(method *protogen.Method) *methodOptions {
	opts := method.Desc.Options()
	if opts == nil {
		return nil
	}

	var mOpts *methodOptions
	opts.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsExtension() && fd.Number() == extMethodOptions {
			mOpts = &methodOptions{}
			msg := v.Message()
			msg.Range(func(subFd protoreflect.FieldDescriptor, subV protoreflect.Value) bool {
				switch subFd.Number() {
				case 1:
					mOpts.ToolName = subV.String()
				case 2:
					mOpts.Description = subV.String()
				case 3:
					mOpts.Skip = subV.Bool()
				case 4:
					mOpts.Resource = subV.Bool()
				case 5:
					mOpts.ResourceURITemplate = subV.String()
				case 6:
					mOpts.ReadOnly = subV.Bool()
				case 7:
					mOpts.Destructive = subV.Bool()
				case 8:
					mOpts.Version = int32(subV.Int())
				case 9:
					mOpts.IsDeprecated = subV.Bool()
				}
				return true
			})
			return false // stop iterating fields
		}
		return true
	})

	return mOpts
}

func readServiceOptions(svc *protogen.Service) *serviceOptions {
	opts := svc.Desc.Options()
	if opts == nil {
		return nil
	}

	var sOpts *serviceOptions
	opts.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsExtension() && fd.Number() == extServiceOptions {
			sOpts = &serviceOptions{}
			msg := v.Message()
			msg.Range(func(subFd protoreflect.FieldDescriptor, subV protoreflect.Value) bool {
				switch subFd.Number() {
				case 1:
					sOpts.ToolNamePrefix = subV.String()
				case 2:
					sOpts.Description = subV.String()
				}
				return true
			})
			return false
		}
		return true
	})

	return sOpts
}

func readFileOptions(file *protogen.File) *fileOptions {
	opts := file.Desc.Options()
	if opts == nil {
		return nil
	}

	var fOpts *fileOptions
	opts.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsExtension() && fd.Number() == extFileOptions {
			fOpts = &fileOptions{}
			msg := v.Message()
			msg.Range(func(subFd protoreflect.FieldDescriptor, subV protoreflect.Value) bool {
				if subFd.Number() == 1 {
					fOpts.Skip = subV.Bool()
				}
				return true
			})
			return false
		}
		return true
	})

	return fOpts
}
