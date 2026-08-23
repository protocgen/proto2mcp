# OSS-Fuzz Integration

This directory contains the files required for integrating `proto2mcp` with Google's [OSS-Fuzz](https://google.github.io/oss-fuzz/).

## Fuzz Targets

The fuzz targets are located in `pkg/mcpruntime/fuzz_test.go` and include:
- `FuzzSanitizeErrorMessage`
- `FuzzTruncateUTF8`
- `FuzzUnmarshalToolInput`
- `FuzzResourceKeyExtraction`

## Submitting to OSS-Fuzz

To submit this project to OSS-Fuzz:
1. Ensure these files are merged into the main branch.
2. Follow the [New Project Guide](https://google.github.io/oss-fuzz/getting-started/new-project-guide/) and submit a PR to the [oss-fuzz repository](https://github.com/google/oss-fuzz).
