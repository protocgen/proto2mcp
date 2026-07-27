# Contributing to proto2mcp

Thank you for your interest in contributing to proto2mcp!

## Getting Started

```bash
git clone https://github.com/protocgen/proto2mcp.git
cd proto2mcp
make ci  # lint + test + build
```

### Project Structure

```text
proto2mcp/
├── codegen/                    # protoc plugin (separate Go module)
│   ├── cmd/protoc-gen-proto2mcp/  # plugin entrypoint
│   └── pkg/
│       ├── extract/            # proto → IR extraction + JSON Schema
│       └── emit/               # IR → Go code generation (jennifer)
├── pkg/mcpruntime/             # runtime library (users import this)
│   └── connectbridge/          # optional ConnectRPC error mapping
├── proto/                      # options proto definitions
├── examples/healthcare/        # working example
└── Makefile                    # build, test, lint, release targets
```

### Two Go Modules

This project uses two Go modules:
- **Root** (`go.mod`): runtime library (`pkg/mcpruntime/`)
- **`codegen/`** (`codegen/go.mod`): protoc plugin with heavy codegen dependencies

This keeps the runtime dependency tree minimal for users.

## Development Workflow

1. **Create a branch** from `main`
2. **Make changes** — run `make ci` locally before pushing
3. **Open a PR** — CI runs lint, test (3 OS × race detector), buf lint, govulncheck
4. **Review & merge**

## Useful Make Targets

| Target | Description |
|---|---|
| `make ci` | Full local CI (lint + test + build) |
| `make test` | Tests with race detector |
| `make test-cover` | Tests with coverage report |
| `make test-fuzz` | Fuzz tests (30s each) |
| `make bench` | Benchmarks |
| `make golden-update` | Regenerate golden test files |
| `make lint-full` | golangci-lint (requires install) |

## Guidelines

- **Tests**: All new functionality needs tests. Use table-driven tests.
- **Golden files**: If code generation output changes, update goldens with `make golden-update`.
- **Comments**: Export all public symbols with GoDoc comments.
- **Dependencies**: Minimize runtime dependencies. Heavy deps go in `codegen/`.

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
