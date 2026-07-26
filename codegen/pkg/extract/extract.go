package extract

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FromPlugin extracts FileIR from a protogen.Plugin.
// This is the primary entry point for the protoc plugin.
func FromPlugin(plugin *protogen.Plugin) ([]FileIR, error) {
	var results []FileIR
	for _, f := range plugin.Files {
		if !f.Generate {
			continue
		}
		fileIR, err := ExtractFile(f)
		if err != nil {
			return nil, fmt.Errorf("extracting %s: %w", f.Desc.Path(), err)
		}
		results = append(results, *fileIR)
	}
	return results, nil
}

// FromDescriptors extracts FileIR from proto file descriptors.
// V3: Used by AI API Gateway for runtime extraction.
func FromDescriptors(files []protoreflect.FileDescriptor) ([]FileIR, error) {
	// TODO: Phase 2/V3 — implement runtime extraction
	return nil, nil
}
