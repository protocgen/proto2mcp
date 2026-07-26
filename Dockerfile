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

# Build statically linked binary from the codegen submodule
RUN cd codegen && CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/protoc-gen-proto2mcp ./cmd/protoc-gen-proto2mcp

# --- Final Image Stage ---
# Distroless: ~2MB, no shell, ca-certificates baked in.
# Static variant works because CGO_ENABLED=0.
FROM gcr.io/distroless/static-debian12

# Copy the binary from builder
COPY --from=builder /bin/protoc-gen-proto2mcp /usr/local/bin/protoc-gen-proto2mcp

# Default entrypoint
ENTRYPOINT ["/usr/local/bin/protoc-gen-proto2mcp"]
