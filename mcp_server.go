package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// MCPToolCall MCP工具调用
type MCPToolCall struct {
	Name      string
	Arguments map[string]interface{}
}

// MCPServer MCP服务器
type MCPServer struct {
	zhipinService *ZhipinService
}

// NewMCPServer 创建MCP服务器
func NewMCPServer(service *ZhipinService) *MCPServer {
	return &MCPServer{
		zhipinService: service,
	}
}

// HandleToolCall 处理工具调用
func (s *MCPServer) HandleToolCall(ctx context.Context, call MCPToolCall) *MCPToolResult {
	logrus.Infof("MCP工具调用: %s", call.Name)

	// 创建AppServer用于调用处理器
	appServer := &AppServer{
		zhipinService: s.zhipinService,
	}

	switch call.Name {
	case "check_login_status":
		return appServer.handleCheckLoginStatus(ctx)
	case "get_login_qrcode":
		return appServer.handleGetLoginQrcode(ctx)
	case "delete_cookies":
		return appServer.handleDeleteCookies(ctx)
	case "search_jobs":
		return appServer.handleSearchJobs(ctx, call.Arguments)
	case "get_job_detail":
		return appServer.handleGetJobDetail(ctx, call.Arguments)
	case "deliver_job":
		return appServer.handleDeliverJob(ctx, call.Arguments)
	case "delivered_list":
		return appServer.handleDeliveredList(ctx, call.Arguments)
	case "batch_deliver":
		return appServer.handleBatchDeliver(ctx, call.Arguments)
	case "start_cron":
		return appServer.handleStartCron(ctx, call.Arguments)
	case "stop_cron":
		return appServer.handleStopCron(ctx, call.Arguments)
	case "get_config":
		return appServer.handleGetConfig(ctx)
	case "update_config":
		return appServer.handleUpdateConfig(ctx, call.Arguments)
	case "get_stats":
		return appServer.handleGetStats(ctx)
	default:
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("未知工具: %s", call.Name),
			}},
			IsError: true,
		}
	}
}

// GetTools 获取工具列表（用于注册）
func (s *MCPServer) GetTools() []Tool {
	return []Tool{
		{
			Name:        "check_login_status",
			Description: "检查当前登录状态",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "get_login_qrcode",
			Description: "获取登录二维码，用于扫码登录BOSS直聘",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "delete_cookies",
			Description: "删除Cookie并重置登录状态",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "search_jobs",
			Description: "搜索职位",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword":    map[string]interface{}{"type": "string", "description": "搜索关键词"},
					"city":       map[string]interface{}{"type": "string", "description": "城市"},
					"district":   map[string]interface{}{"type": "string", "description": "区域"},
					"experience": map[string]interface{}{"type": "string", "description": "经验要求"},
					"education":  map[string]interface{}{"type": "string", "description": "学历要求"},
					"job_type":   map[string]interface{}{"type": "string", "description": "工作类型"},
					"salary":     map[string]interface{}{"type": "string", "description": "薪资范围"},
					"page":       map[string]interface{}{"type": "integer", "description": "页码"},
				},
			},
		},
		{
			Name:        "get_job_detail",
			Description: "获取职位详情",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"job_id": map[string]interface{}{"type": "string", "description": "职位ID"},
				},
				"required": []string{"job_id"},
			},
		},
		{
			Name:        "deliver_job",
			Description: "投递简历到指定职位",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"job_id": map[string]interface{}{"type": "string", "description": "职位ID"},
				},
				"required": []string{"job_id"},
			},
		},
		{
			Name:        "delivered_list",
			Description: "获取已投递职位列表",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit":  map[string]interface{}{"type": "integer", "description": "返回数量限制"},
					"offset": map[string]interface{}{"type": "integer", "description": "偏移量"},
				},
			},
		},
		{
			Name:        "batch_deliver",
			Description: "批量投递简历",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"job_ids": map[string]interface{}{
						"type":        "array",
						"description": "职位ID列表",
						"items":       map[string]interface{}{"type": "string"},
					},
				},
				"required": []string{"job_ids"},
			},
		},
		{
			Name:        "start_cron",
			Description: "启动定时任务",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_name":       map[string]interface{}{"type": "string", "description": "任务名称"},
					"cron_expression": map[string]interface{}{"type": "string", "description": "Cron表达式"},
					"keyword":         map[string]interface{}{"type": "string", "description": "搜索关键词"},
					"city":            map[string]interface{}{"type": "string", "description": "城市"},
				},
				"required": []string{"task_name", "cron_expression", "keyword"},
			},
		},
		{
			Name:        "stop_cron",
			Description: "停止定时任务",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "integer", "description": "任务ID"},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "get_config",
			Description: "获取当前配置",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "update_config",
			Description: "更新配置",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username":  map[string]interface{}{"type": "string", "description": "用户名"},
					"password":  map[string]interface{}{"type": "string", "description": "密码（AES加密后）"},
					"max_daily": map[string]interface{}{"type": "integer", "description": "每日投递上限"},
				},
			},
		},
		{
			Name:        "get_stats",
			Description: "获取投递统计",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}
}

// Tool MCP工具定义
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}
