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
go test -race -count=1 ./...
```

If you modify code generation logic, you may need to update golden files:
```bash
cd codegen && go test -run TestGoldenFiles ./pkg/emit/ -args -update
```

## Protocol Buffers

If you modify any `.proto` files, regenerate the Go code using `buf`:
```bash
buf generate
```

## Code Style

- Follow standard Go conventions.
- Adhere to the Protocol Buffers style guide.
- Keep your commits atomic and well-documented.
