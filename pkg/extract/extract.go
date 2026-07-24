package extract

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FromPlugin extracts ServiceIR from a protogen.Plugin.
// This is the primary entry point for the protoc plugin.
func FromPlugin(plugin *protogen.Plugin) ([]ServiceIR, error) {
	// TODO: Phase 2 — implement descriptor walking
	return nil, nil
}

// FromDescriptors extracts ServiceIR from proto file descriptors.
// V3: Used by AI API Gateway for runtime extraction.
func FromDescriptors(files []protoreflect.FileDescriptor) ([]ServiceIR, error) {
	// TODO: Phase 2/V3 — implement runtime extraction
	return nil, nil
}
