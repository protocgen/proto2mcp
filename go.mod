module github.com/protocgen/proto2mcp

go 1.25.0

require (
	connectrpc.com/connect v1.18.1
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/metric v1.35.0
	google.golang.org/protobuf v1.36.11
)

require buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260709200747-435963d16310.1

require (
	github.com/dave/jennifer v1.7.1
	hegel.dev/go/hegel v0.6.21
)

require (
	github.com/ebitengine/purego v0.11.0-alpha.6.0.20260707033313-5f49e7c49322 // indirect
	golang.org/x/sys v0.44.0 // indirect
)
