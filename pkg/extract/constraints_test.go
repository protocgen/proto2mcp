package extract

import (
	"testing"

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
