# Changelog

## v0.12.3

### Fixed

- **`truncateUTF8` produced invalid UTF-8** when input contained pre-existing invalid bytes (e.g., lone `\xe6` lead byte from untrusted tenant IDs). Found by fuzz testing. Fixed by sanitizing with `strings.ToValidUTF8` before truncation (#126)

### Added

- **Fuzz tests** — 4 fuzz targets covering security-sensitive parsing: `sanitizeErrorMessage`, `truncateUTF8`, `UnmarshalToolInput`, resource key extraction. Crash corpus committed as regression test (#126)
- **Proto-driven quickstart** (`examples/proto-quickstart/`) — full codegen workflow: `.proto` → `buf generate` → implement generated interface → serve. Same TodoService as quickstart for direct comparison (#126)

## v0.12.2

### Added

- **E2E codegen test** — full pipeline: construct IR → `GenerateFile` + `GeneratePrompts` → parse with `go/parser` → assert symbols (#125)
- **Runtime benchmarks** — WrapHandler (34ns), FilteredTools (1.7µs), RateLimiter (224ns), Lookup (26ns/0 allocs), proto marshal/unmarshal, error sanitization (#125)

## v0.12.1

### Added

- **Quickstart example** (`examples/quickstart/`) — runnable MCP server in 3 files, zero external deps (#123)
- **Authorization patterns guide** (`docs/authorization.md`) — 3-tier auth guide with transport/tool layer boundary (#122)
- **Proto Generate CI job** — `buf generate` + compile verification in CI (#121)

## v0.12.0

### Added

- **Prompt template codegen** — `PromptArgument` proto message, `PromptHandler` interface generation, `RegisterXxxPrompts` function generation (#117)
- **ResourceURI emission** — `resource_uri_template` now emitted in generated `ToolDefinition` literals (#117)
- **Annotations emission** — `readOnlyHint` / `destructiveHint` emitted in `ToolDefinition.Annotations` (#117)
- **FilteredTools** — `ToolRegistry.FilteredTools(ctx, ...Middleware)` applies `DiscoveryInterceptor` chain for tool listing (#116)
- **Header allowlist** — `DefaultHeaderAllowlist` + `FilterHeaders` for ConnectRPC forwarders; `WithHeaderAllowlist` option (#116)
- **RateLimiter middleware** — per-tenant-per-tool token bucket with bounded buckets and stale eviction (#119)
- **ErrorMapper** — `VerboseErrors` toggle for production vs development error detail (#119)
- **ResourceKeyValidator** — `WithResourceKeyValidator` option for resource key value validation (#119)
- **NewBoundedMetrics** — bounded cardinality variant of `NewMetrics` with validTools set (#119)
- **SequentialExecutor** — V3 macro-tool executor with fail-fast semantics (#118)
- **Macro codegen** — extract `MacroDefinition` from proto, emit `RegisterMacro` + `MacroStep` (#118)

### Changed

- `NewMetrics` signature preserved for backward compatibility; use `NewBoundedMetrics` for production (#119)
- Generated ConnectRPC forwarders now filter headers through allowlist (#116)
- Tenant ID metrics label capped at 128 bytes with UTF-8 rune safety (#119)

### Improved

- `.golangci.yml` v2 config with conservative linters (#115)
- `CONTRIBUTING.md` with nix develop setup, testing, and style guide (#115)
- `Makefile` split: `make test` (cached, pre-push) / `make test-ci` (no cache) (#115)
- README: CI badge, middleware metadata section (#115)
- ResourceKeys documented as untrusted agent input (#119)
- Parallel macro steps emit warnings (deferred, not silently ignored) (#118)

### Fixed

- Golden test CRLF normalization for Windows CI (#117)

## v0.11.0

See [releases](https://github.com/protocgen/proto2mcp/releases/tag/v0.11.0).
