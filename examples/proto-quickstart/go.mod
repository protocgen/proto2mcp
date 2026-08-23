module github.com/protocgen/proto2mcp/examples/proto-quickstart

go 1.25.0

require (
	connectrpc.com/connect v1.20.0
	github.com/protocgen/proto2mcp v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
)

replace github.com/protocgen/proto2mcp => ../../
