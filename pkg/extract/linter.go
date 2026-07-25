package extract

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// LintMethod checks a proto method for LLM-hostile patterns.
// Returns any warnings found.
func LintMethod(method *protogen.Method) []Warning {
	var warnings []Warning
	methodName := string(method.Desc.Name())

	if method.Desc.IsStreamingClient() && method.Desc.IsStreamingServer() {
		warnings = append(warnings, Warning{
			Severity: WarnError,
			Method:   methodName,
			Message:  "MCP does not support bidirectional streaming",
		})
	} else if method.Desc.IsStreamingClient() {
		warnings = append(warnings, Warning{
			Severity: WarnError,
			Method:   methodName,
			Message:  "MCP does not support client streaming",
		})
	} else if method.Desc.IsStreamingServer() {
		warnings = append(warnings, Warning{
			Severity: WarnError,
			Method:   methodName,
			Message:  "MCP does not support server streaming",
		})
	}

	// For the "description option", since we don't have the explicit extension imported,
	// we assume the primary source of description is the comment.
	if strings.TrimSpace(method.Comments.Leading.String()) == "" {
		warnings = append(warnings, Warning{
			Severity: WarnWarning,
			Method:   methodName,
			Message:  "Method has no description; LLM will not understand its purpose",
		})
	}

	if method.Input != nil {
		if len(method.Input.Fields) == 0 {
			warnings = append(warnings, Warning{
				Severity: WarnWarning,
				Method:   methodName,
				Message:  "Input message has no fields; tool has no parameters",
			})
		}

		if len(method.Input.Fields) > 20 {
			warnings = append(warnings, Warning{
				Severity: WarnWarning,
				Method:   methodName,
				Message:  fmt.Sprintf("Input message has %d fields; large schemas may confuse LLMs", len(method.Input.Fields)),
			})
		}

		msgWarnings := LintMessage(method.Input, 0)
		for _, w := range msgWarnings {
			w.Method = methodName
			warnings = append(warnings, w)
		}
	}

	return warnings
}

// LintMessage checks a proto message for LLM-hostile patterns.
// depth is the current nesting depth (start at 0).
func LintMessage(msg *protogen.Message, depth int) []Warning {
	var warnings []Warning

	if depth > 4 {
		warnings = append(warnings, Warning{
			Severity: WarnWarning,
			Message:  fmt.Sprintf("Message nesting depth %d exceeds recommended maximum of 4", depth),
		})
		return warnings
	}

	for _, field := range msg.Fields {
		if field.Message != nil && string(field.Message.Desc.FullName()) == "google.protobuf.Any" {
			warnings = append(warnings, Warning{
				Severity: WarnError,
				Message:  "google.protobuf.Any cannot be represented as JSON Schema for LLM consumption",
			})
		}

		if strings.TrimSpace(field.Comments.Leading.String()) == "" {
			warnings = append(warnings, Warning{
				Severity: WarnInfo,
				Message:  fmt.Sprintf("Field '%s' has no description; may reduce LLM accuracy", field.Desc.Name()),
			})
		}

		if field.Message != nil {
			if !strings.HasPrefix(string(field.Message.Desc.FullName()), "google.protobuf.") {
				warnings = append(warnings, LintMessage(field.Message, depth+1)...)
			}
		}
	}

	return warnings
}
