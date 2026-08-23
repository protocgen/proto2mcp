package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/protocgen/proto2mcp/pkg/mcpruntime"
)

// jsonRPCRequest represents a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError represents a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// mcpToolListResult matches the MCP tools/list response format.
type mcpToolListResult struct {
	Tools []mcpTool `json:"tools"`
}

// mcpTool matches the MCP tool definition format.
type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Annotations map[string]any  `json:"annotations,omitempty"`
}

// mcpCallParams are the parameters for tools/call.
type mcpCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// mcpCallResult matches the MCP tools/call response format.
type mcpCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// mcpContent is a single content block in a tool result.
type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewMCPHandler creates an HTTP handler that serves the MCP JSON-RPC protocol.
// This is a minimal reference implementation — for production use, consider
// a full MCP framework like mcp-go.
func NewMCPHandler(registry *mcpruntime.ToolRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &jsonRPCError{Code: -32700, Message: "parse error"},
			})
			return
		}

		var resp jsonRPCResponse
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = map[string]any{
				"protocolVersion": "2025-03-26",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]string{
					"name":    "proto2mcp-quickstart",
					"version": "0.1.0",
				},
			}

		case "tools/list":
			defs := registry.Tools()
			tools := make([]mcpTool, len(defs))
			for i, d := range defs {
				tools[i] = mcpTool{
					Name:        d.Name,
					Description: d.Description,
					InputSchema: d.InputSchema,
					Annotations: d.Annotations,
				}
			}
			resp.Result = mcpToolListResult{Tools: tools}

		case "tools/call":
			var params mcpCallParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				resp.Error = &jsonRPCError{Code: -32602, Message: "invalid params"}
				writeJSON(w, resp)
				return
			}

			handler, ok := registry.Lookup(params.Name)
			if !ok {
				resp.Error = &jsonRPCError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", params.Name)}
				writeJSON(w, resp)
				return
			}

			toolReq := mcpruntime.ToolRequest{
				ToolName:  params.Name,
				Arguments: params.Arguments,
			}

			result, err := handler(r.Context(), toolReq)
			if err != nil {
				resp.Error = &jsonRPCError{Code: -32603, Message: "internal error"}
				writeJSON(w, resp)
				return
			}

			resp.Result = mcpCallResult{
				Content: []mcpContent{{
					Type: "text",
					Text: string(result.Content),
				}},
				IsError: result.IsError,
			}

		default:
			resp.Error = &jsonRPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
		}

		writeJSON(w, resp)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
