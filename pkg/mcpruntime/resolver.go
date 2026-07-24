package mcpruntime

import (
	"context"
)

// ServiceResolver dynamically resolves backend service clients.
// V1: Not used (compile-time wiring via ForwardToConnect).
// V3: AI API Gateway uses this for runtime service discovery.
type ServiceResolver interface {
	ResolveEndpoint(ctx context.Context, serviceName string) (string, error)
}
