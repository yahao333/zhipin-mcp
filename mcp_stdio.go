package main

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// MCPStdioServer MCP STDIO 服务器
type MCPStdioServer struct {
	mcpServer *MCPServer
	mu        sync.Mutex
}

// NewMCPStdioServer 创建 STDIO 服务器
func NewMCPStdioServer(mcpServer *MCPServer) *MCPStdioServer {
	return &MCPStdioServer{
		mcpServer: mcpServer,
	}
}

// Run 运行 STDIO 服务器
func (s *MCPStdioServer) Run(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)

	logrus.Info("MCP STDIO 服务器已启动，等待客户端连接...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var rpcReq JSONRPCRequest
		if err := decoder.Decode(&rpcReq); err != nil {
			if err.Error() == "EOF" {
				logrus.Info("STDIO 客户端已断开连接")
				return nil
			}
			logrus.Warnf("解码 JSON-RPC 请求失败: %v", err)
			continue
		}

		// 同步处理请求
		s.handleRequest(rpcReq)
	}
}

// handleRequest 处理 JSON-RPC 请求
func (s *MCPStdioServer) handleRequest(req JSONRPCRequest) JSONRPCResponse {
	var resp JSONRPCResponse

	switch req.Method {
	case "initialize":
		resp = s.handleInitialize(req)
	case "tools/list":
		resp = s.handleToolsList(req)
	case "tools/call":
		resp = s.handleToolsCall(req)
	case "ping":
		resp = s.handlePing(req)
	default:
		// 未知方法返回错误
		resp = s.newErrorResponse(req.ID, -32601, "Method not found")
	}

	// 发送响应
	s.sendResponse(resp)

	return resp
}

// handleInitialize 处理 initialize 请求
func (s *MCPStdioServer) handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	logrus.Info("收到 initialize 请求")

	// 解析初始化参数（可选）
	var params InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			logrus.Warnf("解析 initialize 参数失败: %v", err)
		}
	}

	// 返回初始化结果
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

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// handleToolsList 处理 tools/list 请求
func (s *MCPStdioServer) handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	logrus.Info("收到 tools/list 请求")

	tools := s.mcpServer.GetTools()

	// 转换为 MCP 格式
	mcpTools := make([]ToolDefinition, len(tools))
	for i, tool := range tools {
		mcpTools[i] = ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	result := ToolListResult{
		Tools: mcpTools,
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// handleToolsCall 处理 tools/call 请求
func (s *MCPStdioServer) handleToolsCall(req JSONRPCRequest) JSONRPCResponse {
	logrus.Infof("收到 tools/call 请求: %s", req.Method)

	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.newErrorResponse(req.ID, -32602, "Invalid params")
	}

	if params.Name == "" {
		return s.newErrorResponse(req.ID, -32602, "Missing tool name")
	}

	// 构建工具调用
	call := MCPToolCall{
		Name:      params.Name,
		Arguments: params.Arguments,
	}

	// 调用工具
	ctx := context.Background()
	result := s.mcpServer.HandleToolCall(ctx, call)

	// 转换为 MCP 格式
	toolResult := ToolCallResult{
		Content: result.Content,
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  toolResult,
		ID:      req.ID,
	}
}

// handlePing 处理 ping 请求
func (s *MCPStdioServer) handlePing(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  map[string]string{"status": "pong"},
		ID:      req.ID,
	}
}

// sendResponse 发送 JSON-RPC 响应
func (s *MCPStdioServer) sendResponse(resp JSONRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(resp); err != nil {
		logrus.Warnf("发送响应失败: %v", err)
	}
}

// newErrorResponse 创建错误响应
func (s *MCPStdioServer) newErrorResponse(id interface{}, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id,omitempty"`
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeParams 初始化参数
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion,omitempty"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
	ClientInfo      ClientInfo             `json:"clientInfo,omitempty"`
}

// ClientInfo 客户端信息
type ClientInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Capabilities 服务器能力
type Capabilities struct {
	Tools *ToolCapability `json:"tools,omitempty"`
}

// ToolCapability 工具能力
type ToolCapability struct{}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult 初始化结果
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// ToolListResult 工具列表结果
type ToolListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ToolCallParams 工具调用参数
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []MCPContent `json:"content"`
}
