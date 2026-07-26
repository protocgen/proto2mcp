# Multi-stage Dockerfile for protoc-gen-proto2mcp

# --- Build Stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Copy go.mod and go.sum for caching dependency layers
COPY go.mod go.sum ./
COPY codegen/go.mod codegen/go.sum ./codegen/

RUN go mod download && cd codegen && go mod download

# Copy the rest of the source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/protoc-gen-proto2mcp ./codegen/cmd/protoc-gen-proto2mcp

# --- Final Image Stage ---
# Use a minimal alpine image for execution.
FROM alpine:3.21

# Install basic certificates and libc compatibility if needed.
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /bin/protoc-gen-proto2mcp /usr/local/bin/protoc-gen-proto2mcp

# Default command prints help/version info
ENTRYPOINT ["/usr/local/bin/protoc-gen-proto2mcp"]
