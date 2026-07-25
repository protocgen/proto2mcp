// Package emit generates Go source code from the extract.IR using
// the dave/jennifer code generation library. It produces .pb.mcp.go
// files that contain MCP tool definitions, handler interfaces, and
// registration functions for each protobuf service.
package emit
