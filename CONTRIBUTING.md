# Contributing to proto2mcp

We welcome contributions! This document outlines the process for contributing to the project.

## Prerequisites

The project uses Nix to manage dependencies. This is the recommended way to set up your environment.
- Nix (for `nix develop` shell)

Alternatively, you can manually install the required tools:
- Go 1.25+
- buf
- golangci-lint

## Getting Started

1. Enter the development shell:
   ```bash
   nix develop
   ```
2. Run tests to ensure everything is working:
   ```bash
   make test
   ```
3. Run formatting and linting checks:
   ```bash
   make hygiene
   ```

## Project Structure

```text
proto/                     # Protobuf definitions (options.proto)
pkg/mcpruntime/            # Runtime library (registry, middleware, metrics)
pkg/mcpruntime/connectbridge/  # ConnectRPC error mapping
codegen/                   # protoc-gen-proto2mcp plugin (separate go.mod)
codegen/pkg/extract/       # Proto → IR extraction
codegen/pkg/emit/          # IR → Go code generation
examples/quickstart/       # Zero-dep runnable example
examples/proto-quickstart/ # Full codegen workflow example
examples/connectrpc-bridge/# ConnectRPC forwarding guide
examples/healthcare/       # Healthcare domain example
docs/                      # Authorization guide, etc.
oss-fuzz/                  # Google OSS-Fuzz integration
```

## Branch Conventions

Please use the following prefixes for your branches:
- `feat/` for new features
- `fix/` for bug fixes
- `chore/` for maintenance, hygiene, and tooling changes

## Git Hooks

We use `lefthook` to manage Git hooks.
- **Pre-commit**: Runs `gofmt`, `go vet`, `golangci-lint`, `editorconfig-checker`, and `yamllint`.
- **Pre-push**: Runs `make test` which executes the full test suite with the race detector.

## Testing

The project is split into root and codegen modules. Run tests for both:
```bash
make test-ci    # No cache, race detector
make test       # Cached, for pre-push
```

If you modify code generation logic, you may need to update golden files:
```bash
cd codegen && go test -run TestGoldenFiles ./pkg/emit/ -args -update
```

### Fuzz Testing

We have fuzz tests for security-sensitive parsing functions:
```bash
# Run a specific fuzzer for 30 seconds
go test -fuzz=FuzzSanitizeErrorMessage -fuzztime=30s ./pkg/mcpruntime/
go test -fuzz=FuzzTruncateUTF8 -fuzztime=30s ./pkg/mcpruntime/
go test -fuzz=FuzzUnmarshalToolInput -fuzztime=30s ./pkg/mcpruntime/
go test -fuzz=FuzzResourceKeyExtraction -fuzztime=30s ./pkg/mcpruntime/
```

CI runs each fuzzer for 10s per PR. If you find a crash, commit the corpus entry in `pkg/mcpruntime/testdata/fuzz/` as a regression test.

### Benchmarks

Run benchmarks to check performance impact of your changes:
```bash
go test -bench=. -benchmem -count=3 ./pkg/mcpruntime/
```

## Protocol Buffers

If you modify any `.proto` files, regenerate the Go code using `buf`:
```bash
buf generate
```

Note: `gen/` is gitignored in the root module. Generated code only exists locally after `buf generate`. The codegen module tests cannot import from `gen/`.

## CI

PRs run the following checks:

| Job | What |
|-----|------|
| Build & Vet | `go build`, `go vet` on Linux/macOS/Windows |
| Test | `go test -race` with coverage on all 3 OS |
| Lint | `golangci-lint` (root + codegen) |
| Proto Lint | `buf lint` |
| Proto Generate | Build plugin, `buf generate`, verify compiles |
| Security | `govulncheck` |
| Fuzz | 4 fuzz targets × 10s each |
| File Hygiene | editorconfig, merge markers, large files |

## PR Checklist

- [ ] Tests pass (`make test-ci`)
- [ ] No lint errors (checked by CI)
- [ ] Golden files updated if codegen changed
- [ ] CHANGELOG.md updated for user-facing changes
- [ ] Documentation updated if API changed

## Code Style

- Follow standard Go conventions.
- Adhere to the Protocol Buffers style guide.
- Keep your commits atomic and well-documented.
- Use conventional commit prefixes: `feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`.

