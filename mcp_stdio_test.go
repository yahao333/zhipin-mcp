package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestJSONRPCRequestParse 测试 JSON-RPC 请求解析
func TestJSONRPCRequestParse(t *testing.T) {
	// 测试解析 tools/list 请求
	jsonStr := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	var req JSONRPCRequest
	err := json.Unmarshal([]byte(jsonStr), &req)

	assert.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, "tools/list", req.Method)
	// JSON 解析数字时会解析为 float64
	assert.Equal(t, float64(1), req.ID)
}

// TestJSONRPCRequestParseWithParams 测试带参数的 JSON-RPC 请求解析
func TestJSONRPCRequestParseWithParams(t *testing.T) {
	// 测试解析带参数的请求
	jsonStr := `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`
	var req JSONRPCRequest
	err := json.Unmarshal([]byte(jsonStr), &req)

	assert.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, "initialize", req.Method)
	// JSON 解析数字时会解析为 float64
	assert.Equal(t, float64(2), req.ID)

	// 解析参数
	var params InitializeParams
	err = json.Unmarshal(req.Params, &params)
	assert.NoError(t, err)
	assert.Equal(t, "2024-11-05", params.ProtocolVersion)
	assert.Equal(t, "test", params.ClientInfo.Name)
}

// TestInitializeResult 测试初始化结果序列化
func TestInitializeResult(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: Capabilities{
			Tools: &ToolCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "zhipin-mcp",
			Version: "1.0.0",
		},
	}

	// 序列化为 JSON
	data, err := json.Marshal(result)
	assert.NoError(t, err)

	// 验证序列化结果
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "2024-11-05", parsed["protocolVersion"])
	assert.Equal(t, "zhipin-mcp", parsed["serverInfo"].(map[string]interface{})["name"])
}

// TestToolListResult 测试工具列表结果序列化
func TestToolListResult(t *testing.T) {
	result := ToolListResult{
		Tools: []ToolDefinition{
			{
				Name:        "check_login_status",
				Description: "检查当前登录状态",
			},
			{
				Name:        "search_jobs",
				Description: "搜索职位",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}

	// 序列化为 JSON
	data, err := json.Marshal(result)
	assert.NoError(t, err)

	// 验证序列化结果
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	tools := parsed["tools"].([]interface{})
	assert.Equal(t, 2, len(tools))
}

// TestToolCallParams 测试工具调用参数解析
func TestToolCallParams(t *testing.T) {
	jsonStr := `{"name":"search_jobs","arguments":{"keyword":"Go工程师","city":"北京"}}`
	var params ToolCallParams
	err := json.Unmarshal([]byte(jsonStr), &params)

	assert.NoError(t, err)
	assert.Equal(t, "search_jobs", params.Name)
	assert.Equal(t, "Go工程师", params.Arguments["keyword"])
	assert.Equal(t, "北京", params.Arguments["city"])
}

// TestJSONRPCError 测试错误响应
func TestJSONRPCError(t *testing.T) {
	errorResp := JSONRPCError{
		Code:    -32601,
		Message: "Method not found",
	}

	// 序列化为 JSON
	data, err := json.Marshal(errorResp)
	assert.NoError(t, err)

	// 验证序列化结果
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, float64(-32601), parsed["code"])
	assert.Equal(t, "Method not found", parsed["message"])
}

// TestJSONRPCResponse 测试完整响应序列化
func TestJSONRPCResponse(t *testing.T) {
	// 测试成功响应
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result: map[string]string{
			"status": "pong",
		},
		ID: 1,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", parsed["jsonrpc"])
	assert.Equal(t, "pong", parsed["result"].(map[string]interface{})["status"])
	assert.Equal(t, float64(1), parsed["id"])

	// 测试错误响应
	errorResp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
		},
		ID: 2,
	}

	data, err = json.Marshal(errorResp)
	assert.NoError(t, err)

	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", parsed["jsonrpc"])
	assert.Equal(t, float64(-32601), parsed["error"].(map[string]interface{})["code"])
	assert.Equal(t, "Method not found", parsed["error"].(map[string]interface{})["message"])
}

// TestMCPStdioServerHandleRequest 测试请求处理
func TestMCPStdioServerHandleRequest(t *testing.T) {
	// 创建测试服务器
	mcpServer := NewMCPServer(nil)
	stdioServer := NewMCPStdioServer(mcpServer)

	// 测试 handlePing
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "ping",
		ID:      1,
	}
	resp := stdioServer.handleRequest(req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.Result)
	assert.Nil(t, resp.Error)
}

// TestMCPStdioServerHandleToolsList 测试 tools/list 处理
func TestMCPStdioServerHandleToolsList(t *testing.T) {
	mcpServer := NewMCPServer(nil)
	stdioServer := NewMCPStdioServer(mcpServer)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}
	resp := stdioServer.handleRequest(req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.Result)

	// 验证工具列表
	result := resp.Result.(ToolListResult)
	assert.GreaterOrEqual(t, len(result.Tools), 13) // 应该有13个工具
}

// TestMCPStdioServerHandleInitialize 测试 initialize 处理
func TestMCPStdioServerHandleInitialize(t *testing.T) {
	mcpServer := NewMCPServer(nil)
	stdioServer := NewMCPStdioServer(mcpServer)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
	}
	resp := stdioServer.handleRequest(req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.Result)

	// 验证初始化结果
	result := resp.Result.(InitializeResult)
	assert.Equal(t, "2024-11-05", result.ProtocolVersion)
	assert.Equal(t, "zhipin-mcp", result.ServerInfo.Name)
	assert.Equal(t, "1.0.0", result.ServerInfo.Version)
}

// TestMCPStdioServerUnknownMethod 测试未知方法处理
func TestMCPStdioServerUnknownMethod(t *testing.T) {
	mcpServer := NewMCPServer(nil)
	stdioServer := NewMCPStdioServer(mcpServer)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "unknown/method",
		ID:      1,
	}
	resp := stdioServer.handleRequest(req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Result)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
	assert.Equal(t, "Method not found", resp.Error.Message)
}

// TestMCPStdioServerToolCall 测试工具调用
func TestMCPStdioServerToolCall(t *testing.T) {
	// 注意：这里只测试参数解析，不实际执行工具
	mcpServer := NewMCPServer(nil)
	stdioServer := NewMCPStdioServer(mcpServer)

	// 构建 tools/call 请求
	params := ToolCallParams{
		Name:      "check_login_status",
		Arguments: map[string]interface{}{},
	}
	paramsBytes, _ := json.Marshal(params)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsBytes,
		ID:      1,
	}

	resp := stdioServer.handleRequest(req)
	assert.Equal(t, "2.0", resp.JSONRPC)
	// 由于 zhipinService 为 nil，这里会返回错误结果
}
