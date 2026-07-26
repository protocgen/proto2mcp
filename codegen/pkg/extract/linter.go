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

	// Also lint the output message for Any types and deep nesting.
	if method.Output != nil {
		msgWarnings := LintMessage(method.Output, 0)
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

		// Large enums are token-heavy and confuse LLMs.
		if field.Enum != nil && len(field.Enum.Values) > 20 {
			warnings = append(warnings, Warning{
				Severity: WarnWarning,
				Message:  fmt.Sprintf("Field '%s' has enum with %d values; large enums consume many tokens and may confuse LLMs", field.Desc.Name(), len(field.Enum.Values)),
			})
		}

		if field.Message != nil {
			if field.Desc.IsMap() {
				// Map fields have a synthetic map-entry message. Recurse into
				// the map value type instead to avoid false warnings on key/value.
				for _, mf := range field.Message.Fields {
					if mf.Desc.Number() == 2 && mf.Message != nil {
						if !strings.HasPrefix(string(mf.Message.Desc.FullName()), "google.protobuf.") {
							warnings = append(warnings, LintMessage(mf.Message, depth+1)...)
						}
					}
				}
			} else if !strings.HasPrefix(string(field.Message.Desc.FullName()), "google.protobuf.") {
				warnings = append(warnings, LintMessage(field.Message, depth+1)...)
			}
		}
	}

	// Count required fields. Excessive required fields make tools hard to use.
	requiredCount := 0
	for _, field := range msg.Fields {
		if IsFieldRequired(field.Desc) {
			requiredCount++
		}
	}
	if requiredCount > 10 {
		warnings = append(warnings, Warning{
			Severity: WarnWarning,
			Message:  fmt.Sprintf("Message has %d required fields; excessive required fields increase LLM error rates", requiredCount),
		})
	}

	return warnings
}
