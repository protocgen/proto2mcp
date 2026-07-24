// Package extract provides language-agnostic extraction of MCP tool
// definitions from Protobuf service descriptors.
//
// It is a PUBLIC package designed to be imported by both the protoc
// plugin (codegen time) and the AI API Gateway (runtime, V3).
//
// The package reads protobuf definitions and converts them into an
// Intermediate Representation (IR) that is independent of both the
// source (protoc vs runtime) and the destination (Go vs TS vs JSON).
package extract
