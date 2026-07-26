package extract

import (
	"strings"
	"testing"

	validatepb "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestExtractConstraints_Nil(t *testing.T) {
	msgDesc := (&descriptorpb.FieldOptions{}).ProtoReflect().Descriptor()
	fieldDesc := msgDesc.Fields().Get(0)

	constraints := ExtractConstraints(fieldDesc)
	if constraints != nil {
		t.Errorf("expected nil constraints for standard protobuf field, got %v", constraints)
	}
}

func TestIsFieldRequired_Nil(t *testing.T) {
	msgDesc := (&descriptorpb.FieldOptions{}).ProtoReflect().Descriptor()
	fieldDesc := msgDesc.Fields().Get(0)

	req := IsFieldRequired(fieldDesc)
	if req {
		t.Errorf("expected IsFieldRequired to be false for standard protobuf field, got %v", req)
	}
}

func TestIsFieldRequired_Proto2Required(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Name:   proto.String("test.proto"),
		Syntax: proto.String("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TestMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("req_field"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}
	fileDesc, err := protodesc.NewFile(fd, nil)
	if err != nil {
		t.Fatal(err)
	}

	msgDesc := fileDesc.Messages().Get(0)
	fieldDesc := msgDesc.Fields().Get(0)

	if !IsFieldRequired(fieldDesc) {
		t.Error("expected IsFieldRequired to return true for proto2 required field")
	}
}

func TestExtractConstraints_CELRules(t *testing.T) {
	opts := &descriptorpb.FieldOptions{}
	rules := &validatepb.FieldRules{
		CelExpression: []string{"this > 0"},
		Cel: []*validatepb.Rule{
			{
				Id:         proto.String("my_rule"),
				Expression: proto.String("this < 100"),
				Message:    proto.String("value must be less than 100"),
			},
		},
	}
	proto.SetExtension(opts, validatepb.E_Field, rules)

	fd := &descriptorpb.FileDescriptorProto{
		Name:   proto.String("test.proto"),
		Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TestMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:    proto.String("cel_field"),
						Number:  proto.Int32(1),
						Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:    descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
						Options: opts,
					},
				},
			},
		},
	}

	fileDesc, err := protodesc.NewFile(fd, nil)
	if err != nil {
		t.Fatal(err)
	}

	msgDesc := fileDesc.Messages().Get(0)
	fieldDesc := msgDesc.Fields().Get(0)

	constraints := ExtractConstraints(fieldDesc)
	if constraints == nil {
		t.Fatal("expected non-nil constraints")
	}

	notes, ok := constraints["_constraint_notes"].(string)
	if !ok {
		t.Fatal("expected _constraint_notes to be a string")
	}

	if !strings.Contains(notes, "CEL validation: this > 0") {
		t.Errorf("expected notes to contain CEL expression, got %q", notes)
	}
	if !strings.Contains(notes, "Validation rule: value must be less than 100 (CEL: this < 100)") {
		t.Errorf("expected notes to contain CEL rule message, got %q", notes)
	}
}
