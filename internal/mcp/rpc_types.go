package mcp

const jsonRPCVersion = "2.0"

func rpcSuccess(id any, result map[string]any) map[string]any {
	if result == nil {
		result = map[string]any{}
	}
	return map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"result":  result,
	}
}

func rpcError(id any, code int, message string) map[string]any {
	if message == "" {
		message = ""
	}
	return map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}
