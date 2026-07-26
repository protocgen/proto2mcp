package extract

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	validatepb "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

// ExtractConstraints reads buf.validate annotations from a proto field descriptor
// and returns a map of JSON Schema constraint keywords.
// Returns nil if no constraints are present.
func ExtractConstraints(field protoreflect.FieldDescriptor) map[string]any {
	opts := field.Options()
	if opts == nil || !proto.HasExtension(opts, validatepb.E_Field) {
		return nil
	}

	ext := proto.GetExtension(opts, validatepb.E_Field)
	if ext == nil {
		return nil
	}

	rules, ok := ext.(*validatepb.FieldRules)
	if !ok || rules == nil {
		return nil
	}

	result := make(map[string]any)
	var descriptions []string

	// String constraints
	if s := rules.GetString(); s != nil {
		if s.MinLen != nil {
			result["minLength"] = *s.MinLen
		}
		if s.MaxLen != nil {
			result["maxLength"] = *s.MaxLen
		}
		if s.Pattern != nil {
			result["pattern"] = *s.Pattern
		}
		if s.MinBytes != nil {
			descriptions = append(descriptions, fmt.Sprintf("Minimum %d bytes", *s.MinBytes))
		}
		if s.MaxBytes != nil {
			descriptions = append(descriptions, fmt.Sprintf("Maximum %d bytes", *s.MaxBytes))
		}
		if s.Prefix != nil {
			descriptions = append(descriptions, fmt.Sprintf("Must start with '%s'", *s.Prefix))
		}
		if s.Suffix != nil {
			descriptions = append(descriptions, fmt.Sprintf("Must end with '%s'", *s.Suffix))
		}
		if s.Contains != nil {
			descriptions = append(descriptions, fmt.Sprintf("Must contain '%s'", *s.Contains))
		}

		// Well-known string format rules (oneof WellKnown)
		if s.GetEmail() {
			result["format"] = "email"
		} else if s.GetUri() {
			result["format"] = "uri"
		} else if s.GetUuid() {
			result["format"] = "uuid"
		} else if s.GetHostname() {
			result["format"] = "hostname"
		} else if s.GetIp() {
			descriptions = append(descriptions, "Must be a valid IP address (IPv4 or IPv6)")
		} else if s.GetIpv4() {
			result["format"] = "ipv4"
		} else if s.GetIpv6() {
			result["format"] = "ipv6"
		}
	}

	// Numeric constraints — use Has*() / Get*() for oneof-wrapped fields.
	extractInt32Constraints(rules.GetInt32(), result, &descriptions)
	extractInt64Constraints(rules.GetInt64(), result, &descriptions)
	extractUint32Constraints(rules.GetUint32(), result, &descriptions)
	extractUint64Constraints(rules.GetUint64(), result, &descriptions)
	extractFloatConstraints(rules.GetFloat(), result, &descriptions)
	extractDoubleConstraints(rules.GetDouble(), result, &descriptions)

	// Bool constraints
	if r := rules.GetBool(); r != nil {
		if r.HasConst() {
			result["const"] = r.GetConst()
		}
	}

	// Bytes constraints
	if r := rules.GetBytes(); r != nil {
		if r.MinLen != nil {
			result["minLength"] = *r.MinLen
		}
		if r.MaxLen != nil {
			result["maxLength"] = *r.MaxLen
		}
	}

	// Repeated constraints
	if r := rules.GetRepeated(); r != nil {
		if r.MinItems != nil {
			result["minItems"] = *r.MinItems
		}
		if r.MaxItems != nil {
			result["maxItems"] = *r.MaxItems
		}
		if r.Unique != nil && *r.Unique {
			result["uniqueItems"] = true
		}
	}

	// Map constraints
	if r := rules.GetMap(); r != nil {
		if r.MinPairs != nil {
			result["minProperties"] = *r.MinPairs
		}
		if r.MaxPairs != nil {
			result["maxProperties"] = *r.MaxPairs
		}
	}

	if len(descriptions) > 0 {
		result["description"] = strings.Join(descriptions, "; ")
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// IsFieldRequired checks if a field has buf.validate required=true.
func IsFieldRequired(field protoreflect.FieldDescriptor) bool {
	if field.Cardinality() == protoreflect.Required {
		return true
	}

	opts := field.Options()
	if opts == nil || !proto.HasExtension(opts, validatepb.E_Field) {
		return false
	}

	ext := proto.GetExtension(opts, validatepb.E_Field)
	if ext == nil {
		return false
	}

	rules, ok := ext.(*validatepb.FieldRules)
	if !ok || rules == nil {
		return false
	}

	return rules.GetRequired()
}

// extractInt32Constraints reads Int32Rules and populates JSON Schema constraints.
func extractInt32Constraints(r *validatepb.Int32Rules, result map[string]any, descs *[]string) {
	if r == nil {
		return
	}
	if r.HasGt() {
		result["exclusiveMinimum"] = r.GetGt()
	}
	if r.HasGte() {
		result["minimum"] = r.GetGte()
	}
	if r.HasLt() {
		result["exclusiveMaximum"] = r.GetLt()
	}
	if r.HasLte() {
		result["maximum"] = r.GetLte()
	}
	if r.HasConst() {
		result["const"] = r.GetConst()
	}
	if len(r.GetIn()) > 0 {
		result["enum"] = r.GetIn()
	}
	if len(r.GetNotIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must not be one of: %v", r.GetNotIn()))
	}
}

// extractInt64Constraints reads Int64Rules and populates JSON Schema constraints.
func extractInt64Constraints(r *validatepb.Int64Rules, result map[string]any, descs *[]string) {
	if r == nil {
		return
	}
	if r.HasGt() {
		*descs = append(*descs, fmt.Sprintf("Value must be > %v", r.GetGt()))
	}
	if r.HasGte() {
		*descs = append(*descs, fmt.Sprintf("Value must be >= %v", r.GetGte()))
	}
	if r.HasLt() {
		*descs = append(*descs, fmt.Sprintf("Value must be < %v", r.GetLt()))
	}
	if r.HasLte() {
		*descs = append(*descs, fmt.Sprintf("Value must be <= %v", r.GetLte()))
	}
	if r.HasConst() {
		*descs = append(*descs, fmt.Sprintf("Value must be %v", r.GetConst()))
	}
	if len(r.GetIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must be one of: %v", r.GetIn()))
	}
	if len(r.GetNotIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must not be one of: %v", r.GetNotIn()))
	}
}

// extractUint32Constraints reads UInt32Rules and populates JSON Schema constraints.
func extractUint32Constraints(r *validatepb.UInt32Rules, result map[string]any, descs *[]string) {
	if r == nil {
		return
	}
	if r.HasGt() {
		result["exclusiveMinimum"] = r.GetGt()
	}
	if r.HasGte() {
		result["minimum"] = r.GetGte()
	}
	if r.HasLt() {
		result["exclusiveMaximum"] = r.GetLt()
	}
	if r.HasLte() {
		result["maximum"] = r.GetLte()
	}
	if r.HasConst() {
		result["const"] = r.GetConst()
	}
	if len(r.GetIn()) > 0 {
		result["enum"] = r.GetIn()
	}
	if len(r.GetNotIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must not be one of: %v", r.GetNotIn()))
	}
}

// extractUint64Constraints reads UInt64Rules and populates JSON Schema constraints.
func extractUint64Constraints(r *validatepb.UInt64Rules, result map[string]any, descs *[]string) {
	if r == nil {
		return
	}
	if r.HasGt() {
		*descs = append(*descs, fmt.Sprintf("Value must be > %v", r.GetGt()))
	}
	if r.HasGte() {
		*descs = append(*descs, fmt.Sprintf("Value must be >= %v", r.GetGte()))
	}
	if r.HasLt() {
		*descs = append(*descs, fmt.Sprintf("Value must be < %v", r.GetLt()))
	}
	if r.HasLte() {
		*descs = append(*descs, fmt.Sprintf("Value must be <= %v", r.GetLte()))
	}
	if r.HasConst() {
		*descs = append(*descs, fmt.Sprintf("Value must be %v", r.GetConst()))
	}
	if len(r.GetIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must be one of: %v", r.GetIn()))
	}
	if len(r.GetNotIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must not be one of: %v", r.GetNotIn()))
	}
}

// extractFloatConstraints reads FloatRules and populates JSON Schema constraints.
func extractFloatConstraints(r *validatepb.FloatRules, result map[string]any, descs *[]string) {
	if r == nil {
		return
	}
	if r.HasGt() {
		result["exclusiveMinimum"] = r.GetGt()
	}
	if r.HasGte() {
		result["minimum"] = r.GetGte()
	}
	if r.HasLt() {
		result["exclusiveMaximum"] = r.GetLt()
	}
	if r.HasLte() {
		result["maximum"] = r.GetLte()
	}
	if r.HasConst() {
		result["const"] = r.GetConst()
	}
	if len(r.GetIn()) > 0 {
		result["enum"] = r.GetIn()
	}
	if len(r.GetNotIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must not be one of: %v", r.GetNotIn()))
	}
}

// extractDoubleConstraints reads DoubleRules and populates JSON Schema constraints.
func extractDoubleConstraints(r *validatepb.DoubleRules, result map[string]any, descs *[]string) {
	if r == nil {
		return
	}
	if r.HasGt() {
		result["exclusiveMinimum"] = r.GetGt()
	}
	if r.HasGte() {
		result["minimum"] = r.GetGte()
	}
	if r.HasLt() {
		result["exclusiveMaximum"] = r.GetLt()
	}
	if r.HasLte() {
		result["maximum"] = r.GetLte()
	}
	if r.HasConst() {
		result["const"] = r.GetConst()
	}
	if len(r.GetIn()) > 0 {
		result["enum"] = r.GetIn()
	}
	if len(r.GetNotIn()) > 0 {
		*descs = append(*descs, fmt.Sprintf("Must not be one of: %v", r.GetNotIn()))
	}
}
