# Changelog

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
