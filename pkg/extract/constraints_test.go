package extract

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestExtractConstraints_Nil(t *testing.T) {
	// A standard google.protobuf message field won't have buf.validate annotations
	msgDesc := (&descriptorpb.FieldOptions{}).ProtoReflect().Descriptor()
	fieldDesc := msgDesc.Fields().Get(0)

	if fieldDesc == nil {
		t.Fatal("expected to find a field in FieldOptions")
	}

	constraints := ExtractConstraints(fieldDesc)
	if constraints != nil {
		t.Errorf("expected nil constraints for standard protobuf field, got %v", constraints)
	}
}

func TestIsFieldRequired_Nil(t *testing.T) {
	msgDesc := (&descriptorpb.FieldOptions{}).ProtoReflect().Descriptor()
	fieldDesc := msgDesc.Fields().Get(0)

	if fieldDesc == nil {
		t.Fatal("expected to find a field in FieldOptions")
	}

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
