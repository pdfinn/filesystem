package handlers

import (
	"fmt"
	"strings"

	"filesystem/pkg/filesystem"
	"github.com/mark3labs/mcp-go/mcp"
)

// getArguments extracts the argument map from the request and validates its presence.
func getArguments(req mcp.CallToolRequest) (map[string]interface{}, *mcp.CallToolResult) {
	args := req.Params.Arguments
	if args == nil {
		return nil, mcp.NewToolResultError("Invalid arguments format")
	}
	return args, nil
}

// getRequiredString extracts a required string parameter from the argument map.
func getRequiredString(args map[string]interface{}, key string) (string, *mcp.CallToolResult) {
	if val, ok := args[key].(string); ok && val != "" {
		return val, nil
	}
	msg := fmt.Sprintf("%s parameter is required", strings.Title(key))
	return "", mcp.NewToolResultError(msg)
}

// getRequiredStringSlice extracts a required string slice from the argument map.
func getRequiredStringSlice(args map[string]interface{}, key string) ([]string, *mcp.CallToolResult) {
	raw, ok := args[key].([]interface{})
	if !ok {
		msg := fmt.Sprintf("%s parameter is required", strings.Title(key))
		return nil, mcp.NewToolResultError(msg)
	}
	result := make([]string, 0, len(raw))
	for i := 0; i < len(raw) && i < 100; i++ {
		if s, ok := raw[i].(string); ok && s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		msg := fmt.Sprintf("%s parameter is required", strings.Title(key))
		return nil, mcp.NewToolResultError(msg)
	}
	return result, nil
}

// getOptionalStringSlice extracts an optional string slice from the argument map.
func getOptionalStringSlice(args map[string]interface{}, key string) []string {
	raw, ok := args[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for i := 0; i < len(raw) && i < 100; i++ {
		if s, ok := raw[i].(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// getOptionalBool extracts an optional bool parameter with default value.
func getOptionalBool(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// getEditOperations parses edit operations from the argument map.
func getEditOperations(args map[string]interface{}) ([]filesystem.EditOperation, *mcp.CallToolResult) {
	raw, ok := args["edits"].([]interface{})
	if !ok {
		return nil, mcp.NewToolResultError("Edits parameter is required")
	}
	edits := make([]filesystem.EditOperation, 0, len(raw))
	for i := 0; i < len(raw) && i < 100; i++ {
		m, ok := raw[i].(map[string]interface{})
		if !ok {
			continue
		}
		oldText, ok1 := m["oldText"].(string)
		newText, ok2 := m["newText"].(string)
		if ok1 && ok2 {
			edits = append(edits, filesystem.EditOperation{OldText: oldText, NewText: newText})
		}
	}
	if len(edits) == 0 {
		return nil, mcp.NewToolResultError("No valid edits provided")
	}
	return edits, nil
}
