package extract

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// MCP tool name constraints per specification.
const (
	// MaxToolNameLength is the maximum allowed tool name length per MCP spec.
	MaxToolNameLength = 64
)

// toolNamePattern matches valid MCP tool names: alphanumeric, underscores, hyphens.
var toolNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateToolName checks a tool name against MCP spec constraints.
// Returns warnings for any violations.
func ValidateToolName(name string) []Warning {
	var warnings []Warning

	if name == "" {
		warnings = append(warnings, Warning{
			Severity: WarnError,
			Method:   name,
			Message:  "tool name is empty",
		})
		return warnings
	}

	// Note: len(name) counts bytes, not runes. This is intentional and safe:
	// the regex check below rejects any non-ASCII characters, so for all valid
	// names bytes == runes. For invalid names, byte-counting is strictly
	// conservative (overestimates length), which is the right direction.
	if len(name) > MaxToolNameLength {
		warnings = append(warnings, Warning{
			Severity: WarnError,
			Method:   name,
			Message:  fmt.Sprintf("tool name %q exceeds MCP maximum of %d characters (got %d)", name, MaxToolNameLength, len(name)),
		})
	}

	if !toolNamePattern.MatchString(name) {
		warnings = append(warnings, Warning{
			Severity: WarnError,
			Method:   name,
			Message:  fmt.Sprintf("tool name %q contains invalid characters; must match [a-zA-Z0-9_-]", name),
		})
	}

	return warnings
}

// Extension field numbers for MCP annotations.
// Source: proto/protocgen/mcp/v1/options.proto
const (
	// extMethodOptions is the field number for MethodMCPOptions on google.protobuf.MethodOptions.
	extMethodOptions protoreflect.FieldNumber = 1179
	// extServiceOptions is the field number for ServiceMCPOptions on google.protobuf.ServiceOptions.
	extServiceOptions protoreflect.FieldNumber = 1180
	// extFileOptions is the field number for FileMCPOptions on google.protobuf.FileOptions.
	extFileOptions protoreflect.FieldNumber = 1181
	// extFieldOptions is the field number for FieldMCPOptions on google.protobuf.FieldOptions.
	extFieldOptions protoreflect.FieldNumber = 1182
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

	if fOpts != nil {
		ir.Prompts = fOpts.Prompts
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
				ToolNamePrefix:  svcOpts.ToolNamePrefix,
				Description:     svcOpts.Description,
				GenerateConnect: svcOpts.GenerateConnect,
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

	// Validate tool name against MCP spec.
	nameWarnings := ValidateToolName(toolName)

	tool := &ToolIR{
		Name:           toolName,
		MethodName:     string(method.Desc.Name()),
		Description:    desc,
		InputTypeName:  string(method.Input.Desc.FullName()),
		OutputTypeName: string(method.Output.Desc.FullName()),
		IsDeprecated:   isDeprecated,
		Warnings:       nameWarnings,
	}

	if mOpts != nil {
		tool.IsResource = mOpts.Resource
		tool.ResourceURI = mOpts.ResourceURITemplate
		tool.IsReadOnly = mOpts.ReadOnly
		tool.IsDestructive = mOpts.Destructive
		tool.Version = mOpts.Version

		if len(mOpts.Steps) > 0 {
			tool.SubTools = mOpts.Steps
			tool.MacroType = MacroTypeSequential

			// Per stakeholder consensus: warn on parallel, don't silently ignore
			for _, step := range mOpts.Steps {
				if step.Parallel {
					tool.Warnings = append(tool.Warnings, Warning{
						Severity: WarnWarning,
						Method:   tool.Name,
						Message:  fmt.Sprintf("parallel macro step %q is not yet supported; step will run sequentially", step.ToolName),
					})
				}
			}
		}
	}

	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		tool.Warnings = append(tool.Warnings, Warning{
			Severity: WarnError,
			Method:   toolName,
			Message:  "streaming methods are not supported",
		})
	}

	// Generate JSON Schema for the input message.
	if method.Input != nil {
		schema, err := MessageToSchema(method.Input)
		if err != nil {
			tool.Warnings = append(tool.Warnings, Warning{
				Severity: WarnError,
				Method:   toolName,
				Message:  fmt.Sprintf("failed to generate input schema: %v", err),
			})
		} else {
			tool.InputSchema = schema
		}

		// Extract resource keys from field annotations.
		tool.ResourceKeys, tool.Warnings = extractResourceKeys(
			method.Input, toolName, tool.Warnings,
		)
	}

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
	Steps               []ToolRef
}

type serviceOptions struct {
	ToolNamePrefix  string
	Description     string
	GenerateConnect bool
}

type fileOptions struct {
	Skip    bool
	Prompts []PromptIR
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
				case 15: // MacroDefinition message
					macroMsg := subV.Message()
					macroMsg.Range(func(macroFd protoreflect.FieldDescriptor, macroV protoreflect.Value) bool {
						switch macroFd.Number() {
						case 1: // repeated MacroStep steps
							stepList := macroV.List()
							for i := 0; i < stepList.Len(); i++ {
								stepMsg := stepList.Get(i).Message()
								step := ToolRef{}
								stepMsg.Range(func(stepFd protoreflect.FieldDescriptor, stepV protoreflect.Value) bool {
									switch stepFd.Number() {
									case 1: // tool name
										step.ToolName = stepV.String()
									case 2: // parallel
										step.Parallel = stepV.Bool()
									case 3: // output_key
										step.OutputKey = stepV.String()
									}
									return true
								})
								mOpts.Steps = append(mOpts.Steps, step)
							}
						}
						return true
					})
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
				case 3:
					sOpts.GenerateConnect = subV.Bool()
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
				switch subFd.Number() {
				case 1:
					fOpts.Skip = subV.Bool()
				case 10:
					list := subV.List()
					for i := 0; i < list.Len(); i++ {
						item := list.Get(i).Message()
						var p PromptIR
						item.Range(func(pFd protoreflect.FieldDescriptor, pV protoreflect.Value) bool {
							switch pFd.Number() {
							case 1:
								p.Name = pV.String()
							case 2:
								p.Description = pV.String()
							case 3:
								tList := pV.List()
								for j := 0; j < tList.Len(); j++ {
									p.Tools = append(p.Tools, tList.Get(j).String())
								}
							case 4:
								aList := pV.List()
								for j := 0; j < aList.Len(); j++ {
									argItem := aList.Get(j).Message()
									var arg PromptArgIR
									argItem.Range(func(aFd protoreflect.FieldDescriptor, aV protoreflect.Value) bool {
										switch aFd.Number() {
										case 1:
											arg.Name = aV.String()
										case 2:
											arg.Description = aV.String()
										case 3:
											arg.Required = aV.Bool()
										}
										return true
									})
									p.Arguments = append(p.Arguments, arg)
								}
							}
							return true
						})
						fOpts.Prompts = append(fOpts.Prompts, p)
					}
				}
				return true
			})
			return false
		}
		return true
	})

	return fOpts
}

// extractResourceKeys walks the fields of the input message and returns
// the JSON (proto) names of fields annotated with resource_key = true.
// Non-string fields annotated with resource_key generate a linter warning.
func extractResourceKeys(msg *protogen.Message, toolName string, warnings []Warning) ([]string, []Warning) {
	var keys []string
	for _, field := range msg.Fields {
		opts := field.Desc.Options()
		if opts == nil {
			continue
		}

		var isResourceKey bool
		opts.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if fd.IsExtension() && fd.Number() == extFieldOptions {
				subMsg := v.Message()
				subMsg.Range(func(subFd protoreflect.FieldDescriptor, subV protoreflect.Value) bool {
					if subFd.Number() == 1 { // resource_key
						isResourceKey = subV.Bool()
					}
					return true
				})
				return false
			}
			return true
		})

		if !isResourceKey {
			continue
		}

		// Repeated and map fields are not supported as resource keys.
		if field.Desc.IsList() || field.Desc.IsMap() {
			warnings = append(warnings, Warning{
				Severity: WarnWarning,
				Method:   toolName,
				Message:  fmt.Sprintf("resource_key on repeated/map field %q is not supported; only singular string fields can be resource keys", field.Desc.Name()),
			})
			continue
		}

		// Only string fields are supported for V1.
		if field.Desc.Kind() != protoreflect.StringKind {
			warnings = append(warnings, Warning{
				Severity: WarnWarning,
				Method:   toolName,
				Message:  fmt.Sprintf("resource_key on non-string field %q is not supported; only string fields can be resource keys", field.Desc.Name()),
			})
			continue
		}

		keys = append(keys, string(field.Desc.JSONName()))
	}
	return keys, warnings
}
