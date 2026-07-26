package extract

import (
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// buildTestPlugin creates a protogen.Plugin from programmatically created file descriptors.
func buildTestPlugin(t *testing.T, files ...*descriptorpb.FileDescriptorProto) *protogen.Plugin {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{}
	for _, fd := range files {
		req.ProtoFile = append(req.ProtoFile, fd)
		req.FileToGenerate = append(req.FileToGenerate, fd.GetName())
	}
	
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("failed to create protogen.Plugin: %v", err)
	}
	return plugin
}
