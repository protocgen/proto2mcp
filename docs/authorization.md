# Authorization Guide

proto2mcp generates the **tool layer** — registration, schemas, middleware, and handlers. It does **not** handle transport-level authentication (OAuth 2.1, API keys at the HTTP layer, session management). Use your MCP server framework for that.

```
┌─────────────────────────────────────────────────────────────┐
│ Transport Layer (your MCP server framework)                 │
│  • OAuth 2.1 token exchange                                 │
│  • Session management                                       │
│  • HTTP/SSE/Streamable HTTP                                 │
│  • Answers: "Who is this caller?"                           │
└──────────────────────────┬──────────────────────────────────┘
                           │ authenticated request
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ proto2mcp Tool Layer (mcpruntime)                           │
│  • Tool registry + FilteredTools                            │
│  • Middleware chain (authz, rate limiting, logging)          │
│  • Handler execution                                        │
│  • Answers: "Can this caller use THIS tool with THESE args?"│
└─────────────────────────────────────────────────────────────┘
```

This guide covers three authorization patterns at the tool layer, from simple to advanced. Pick the tier that matches your threat model.

---

## Tier 1: Tool Allowlist

**Use when:** You have a few agent roles with fixed tool permissions. No per-request tokens.

```go
// allowlistMiddleware restricts which tools an agent can see and call.
type allowlistMiddleware struct {
    allowed map[string]bool
}

func (a *allowlistMiddleware) HandleToolCall(
    ctx context.Context,
    req mcpruntime.ToolRequest,
    next mcpruntime.HandlerFunc,
) (*mcpruntime.CallToolResult, error) {
    if !a.allowed[req.ToolName] {
        return mcpruntime.InternalError("tool not permitted"), nil
    }
    return next(ctx, req)
}

func (a *allowlistMiddleware) FilterTools(
    ctx context.Context,
    tools []mcpruntime.ToolDefinition,
) []mcpruntime.ToolDefinition {
    var filtered []mcpruntime.ToolDefinition
    for _, t := range tools {
        if a.allowed[t.Name] {
            filtered = append(filtered, t)
        }
    }
    return filtered
}
```

Usage:

```go
readOnly := &allowlistMiddleware{
    allowed: map[string]bool{
        "PatientService_GetPatient":  true,
        "PatientService_ListPatients": true,
    },
}

registry := mcpruntime.NewToolRegistry()
RegisterPatientServiceMCP(registry, handler,
    mcpruntime.WithMiddleware(readOnly),
)

// tools/list only returns permitted tools
tools := registry.FilteredTools(ctx, readOnly)
```

**Pros:** Dead simple, zero dependencies, easy to audit.
**Cons:** Static — can't scope by resource, time, or caller identity.

---

## Tier 2: Token-Based (JWT / API Key)

**Use when:** You need per-caller scoping. Your transport layer passes an identity token that the tool layer inspects.

```go
// jwtAuthzMiddleware checks JWT claims against tool permissions.
type jwtAuthzMiddleware struct {
    // permissions maps role → allowed tool names
    permissions map[string]map[string]bool
}

func (j *jwtAuthzMiddleware) HandleToolCall(
    ctx context.Context,
    req mcpruntime.ToolRequest,
    next mcpruntime.HandlerFunc,
) (*mcpruntime.CallToolResult, error) {
    // Role was extracted by TenantExtractor from the JWT
    role := mcpruntime.TenantFromContext(ctx)

    allowed, ok := j.permissions[role]
    if !ok || !allowed[req.ToolName] {
        return mcpruntime.InternalError("access denied"), nil
    }
    return next(ctx, req)
}

func (j *jwtAuthzMiddleware) FilterTools(
    ctx context.Context,
    tools []mcpruntime.ToolDefinition,
) []mcpruntime.ToolDefinition {
    role := mcpruntime.TenantFromContext(ctx)
    allowed := j.permissions[role]

    var filtered []mcpruntime.ToolDefinition
    for _, t := range tools {
        if allowed[t.Name] {
            filtered = append(filtered, t)
        }
    }
    return filtered
}
```

Usage:

```go
authz := &jwtAuthzMiddleware{
    permissions: map[string]map[string]bool{
        "nurse": {
            "PatientService_GetPatient":   true,
            "PatientService_ListPatients": true,
        },
        "doctor": {
            "PatientService_GetPatient":    true,
            "PatientService_ListPatients":  true,
            "PatientService_UpdatePatient": true,
        },
    },
}

config := mcpruntime.NewConfig(
    mcpruntime.WithTenantExtractor(extractRoleFromJWT),
    mcpruntime.WithMiddleware(authz),
    mcpruntime.WithToolRegistry(registry),
)
```

**Pros:** Per-caller scoping, integrates with existing identity systems.
**Cons:** Permissions are server-side policy — the token itself doesn't encode what's allowed. Can't delegate with attenuation.

---

## Tier 3: Capability Tokens (Macaroons / Biscuit)

**Use when:** You need **delegation chains** — an orchestrator gives an agent a token, the agent narrows it for sub-agents, and each level can only reduce permissions, never expand them.

This is the pattern for multi-agent systems where Agent A spawns Agent B with a subset of its own permissions.

```
Orchestrator
  │  mints token: tools={Get,List,Update}, patient_id=P-123, expires=1h
  ▼
Agent A
  │  attenuates: tools={Get,List}  (dropped Update)
  ▼
Agent B
  │  attenuates: tools={Get}  (dropped List)
  ▼
MCP Server verifies: tool=Get, patient_id=P-123, not expired ✅
```

### Option A: Macaroons (`gopkg.in/macaroon.v2`)

HMAC-based, shared-secret verification. Mature, stable, minimal API.

```go
import "gopkg.in/macaroon.v2"

type macaroonAuthz struct {
    rootKey []byte
}

func (m *macaroonAuthz) HandleToolCall(
    ctx context.Context,
    req mcpruntime.ToolRequest,
    next mcpruntime.HandlerFunc,
) (*mcpruntime.CallToolResult, error) {
    mac := extractMacaroon(ctx) // from Authorization header
    if mac == nil {
        return mcpruntime.InternalError("authorization required"), nil
    }

    // Verify HMAC signature + check all caveats
    err := mac.Verify(m.rootKey, func(caveat string) error {
        parts := strings.SplitN(caveat, " = ", 2)
        if len(parts) != 2 {
            return fmt.Errorf("malformed caveat")
        }
        switch parts[0] {
        case "tool":
            if parts[1] != req.ToolName {
                return fmt.Errorf("tool not permitted")
            }
        default:
            // Check resource keys
            if v, ok := req.ResourceKeys[parts[0]]; ok && v != parts[1] {
                return fmt.Errorf("resource mismatch")
            }
        }
        return nil
    }, nil)

    if err != nil {
        return mcpruntime.InternalError("access denied"), nil
    }
    return next(ctx, req)
}
```

**Trade-off:** Requires shared root key between minter and verifier.

### Option B: Biscuit (`eclipse-biscuit/biscuit-go`)

Public-key crypto with Datalog-based policies. The modern evolution of macaroons.

```go
import biscuit "github.com/biscuit-auth/biscuit-go/v2"

type biscuitAuthz struct {
    publicKey ed25519.PublicKey
}

func (b *biscuitAuthz) HandleToolCall(
    ctx context.Context,
    req mcpruntime.ToolRequest,
    next mcpruntime.HandlerFunc,
) (*mcpruntime.CallToolResult, error) {
    token := extractBiscuit(ctx) // from Authorization header

    // Build authorizer with request facts
    authorizer, _ := token.Authorizer(b.publicKey)
    authorizer.AddFact(biscuit.Fact{
        Predicate: biscuit.Predicate{
            Name: "tool",
            IDs:  []biscuit.Term{biscuit.String(req.ToolName)},
        },
    })
    for k, v := range req.ResourceKeys {
        authorizer.AddFact(biscuit.Fact{
            Predicate: biscuit.Predicate{
                Name: "resource",
                IDs:  []biscuit.Term{biscuit.String(k), biscuit.String(v)},
            },
        })
    }

    // Policy check — Datalog rules in the token determine access
    if err := authorizer.Authorize(); err != nil {
        return mcpruntime.InternalError("access denied"), nil
    }
    return next(ctx, req)
}
```

**Trade-off:** More complex, but public-key verification (no shared secrets), richer policy language, multi-language ecosystem.

### Which to choose?

| Factor | Macaroons | Biscuit |
|--------|-----------|---------|
| Simplicity | ✅ ~30 lines | ⚠️ ~50 lines + Datalog |
| Shared secret required | Yes (HMAC) | No (Ed25519) |
| Policy expressiveness | String matching | Datalog rules |
| Multi-language | Go only | Rust, Go, Java, Haskell |
| Ecosystem maturity | Stable/frozen | Active (Eclipse Foundation) |

For most teams starting out: **macaroons**. For production multi-agent systems with cross-language requirements: **Biscuit**.

---

## Combining Tiers

These patterns compose. A typical production setup:

```go
registry := mcpruntime.NewToolRegistry()
RegisterPatientServiceMCP(registry, handler,
    mcpruntime.WithTenantExtractor(extractFromOAuth),   // transport identity
    mcpruntime.WithMiddleware(
        capabilityAuthz,                                 // tier 3: token scoping
        mcpruntime.NewRateLimiter(10.0, 20),             // rate limiting
        loggingMiddleware,                                // audit trail
    ),
    mcpruntime.WithToolRegistry(registry),
    mcpruntime.WithResourceKeyValidator(validateFormat),  // input safety
)

// tools/list respects all middleware
tools := registry.FilteredTools(ctx, capabilityAuthz)
```

---

## Security Checklist

- [ ] **Transport auth** — OAuth 2.1 or equivalent at the HTTP layer
- [ ] **Tool scoping** — `FilteredTools` ensures agents only see permitted tools
- [ ] **Resource validation** — `WithResourceKeyValidator` rejects malformed keys
- [ ] **Rate limiting** — `NewRateLimiter` prevents agent retry loops
- [ ] **Header filtering** — `DefaultHeaderAllowlist` prevents header leakage to backends
- [ ] **Error verbosity** — `ErrorMapper{VerboseErrors: false}` in production
- [ ] **Metrics bounded** — `NewBoundedMetrics` prevents cardinality explosion
- [ ] **Audit logging** — middleware logs tool calls with tenant + resource keys
