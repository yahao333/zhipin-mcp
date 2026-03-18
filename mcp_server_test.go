package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMCPServer 测试创建 MCP 服务器
func TestNewMCPServer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	service := NewZhipinService()
	server := NewMCPServer(service)

	require.NotNil(t, server)
	assert.NotNil(t, server.zhipinService)
}

// TestMCPServer_GetTools 测试获取工具列表
func TestMCPServer_GetTools(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	tools := server.GetTools()

	// 验证工具列表包含所有必要的工具
	assert.GreaterOrEqual(t, len(tools), 10)

	// 验证关键工具存在
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	// 核心工具
	assert.True(t, toolNames["check_login_status"])
	assert.True(t, toolNames["get_login_qrcode"])
	assert.True(t, toolNames["delete_cookies"])
	assert.True(t, toolNames["search_jobs"])
	assert.True(t, toolNames["get_job_detail"])
	assert.True(t, toolNames["deliver_job"])
	assert.True(t, toolNames["delivered_list"])
	assert.True(t, toolNames["batch_deliver"])
	assert.True(t, toolNames["start_cron"])
	assert.True(t, toolNames["stop_cron"])
	assert.True(t, toolNames["get_config"])
	assert.True(t, toolNames["update_config"])
	assert.True(t, toolNames["get_stats"])
}

// TestTool_Structure 测试工具结构
func TestTool_Structure(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "测试工具",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "参数1",
				},
			},
		},
	}

	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "测试工具", tool.Description)
	assert.NotNil(t, tool.InputSchema)
}

// TestMCPServer_HandleToolCall_UnknownTool 测试处理未知工具
func TestMCPServer_HandleToolCall_UnknownTool(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name:      "unknown_tool",
		Arguments: map[string]interface{}{},
	}

	result := server.HandleToolCall(ctx, call)

	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content)
	assert.Contains(t, result.Content[0].Text, "未知工具")
}

// TestMCPServer_HandleToolCall_CheckLoginStatus 测试处理检查登录状态工具
// 注意：此测试会尝试创建浏览器，可能需要较长时间或超时
func TestMCPServer_HandleToolCall_CheckLoginStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name:      "check_login_status",
		Arguments: map[string]interface{}{},
	}

	result := server.HandleToolCall(ctx, call)

	// 结果不应为nil
	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_GetLoginQrcode 测试处理获取二维码工具
// 注意：此测试会尝试创建浏览器，可能需要较长时间或超时
func TestMCPServer_HandleToolCall_GetLoginQrcode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// 添加超时控制，避免测试无限等待
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server := NewMCPServer(NewZhipinService())

	call := MCPToolCall{
		Name:      "get_login_qrcode",
		Arguments: map[string]interface{}{},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_DeleteCookies 测试处理删除cookies工具
func TestMCPServer_HandleToolCall_DeleteCookies(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name:      "delete_cookies",
		Arguments: map[string]interface{}{},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
	// 删除cookies不应该有错误
	assert.False(t, result.IsError)
}

// TestMCPServer_HandleToolCall_SearchJobs 测试处理搜索职位工具
// 注意：此测试会尝试创建浏览器，可能需要较长时间或超时
func TestMCPServer_HandleToolCall_SearchJobs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "search_jobs",
		Arguments: map[string]interface{}{
			"keyword": "工程师",
			"city":    "北京",
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_GetJobDetail 测试处理获取职位详情工具
// 注意：此测试会尝试创建浏览器，可能需要较长时间或超时
func TestMCPServer_HandleToolCall_GetJobDetail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "get_job_detail",
		Arguments: map[string]interface{}{
			"job_id": "test-job-123",
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_DeliverJob 测试处理投递职位工具
// 注意：此测试会尝试创建浏览器，可能需要较长时间或超时
func TestMCPServer_HandleToolCall_DeliverJob(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "deliver_job",
		Arguments: map[string]interface{}{
			"job_id": "test-job-456",
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_DeliveredList 测试处理已投递列表工具
func TestMCPServer_HandleToolCall_DeliveredList(t *testing.T) {
	// 这个测试需要数据库，让我们跳过它或者修改为只测试参数处理
	t.Skip("需要数据库环境")
}

// TestMCPServer_HandleToolCall_BatchDeliver 测试处理批量投递工具
// 注意：此测试会尝试创建浏览器，可能需要较长时间或超时
func TestMCPServer_HandleToolCall_BatchDeliver(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "batch_deliver",
		Arguments: map[string]interface{}{
			"job_ids": []interface{}{"job-1", "job-2", "job-3"},
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_StartCron 测试处理启动定时任务工具
func TestMCPServer_HandleToolCall_StartCron(t *testing.T) {
	// 需要数据库环境
	t.Skip("需要数据库环境")
}

// TestMCPServer_HandleToolCall_StopCron 测试处理停止定时任务工具
func TestMCPServer_HandleToolCall_StopCron(t *testing.T) {
	// 需要数据库环境
	t.Skip("需要数据库环境")
}

// TestMCPServer_HandleToolCall_GetConfig 测试处理获取配置工具
func TestMCPServer_HandleToolCall_GetConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name:      "get_config",
		Arguments: map[string]interface{}{},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
	// 应该成功获取配置
	assert.False(t, result.IsError)
}

// TestMCPServer_HandleToolCall_UpdateConfig 测试处理更新配置工具
func TestMCPServer_HandleToolCall_UpdateConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "update_config",
		Arguments: map[string]interface{}{
			"max_daily": float64(50),
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
	// 更新配置应该成功
	assert.False(t, result.IsError)
}

// TestMCPServer_HandleToolCall_GetStats 测试处理获取统计工具
func TestMCPServer_HandleToolCall_GetStats(t *testing.T) {
	// 需要数据库
	t.Skip("需要数据库环境")
}

// TestMCPToolCall_Structure 测试工具调用结构
func TestMCPToolCall_Structure(t *testing.T) {
	call := MCPToolCall{
		Name: "test_call",
		Arguments: map[string]interface{}{
			"param1": "value1",
			"param2": float64(123),
		},
	}

	assert.Equal(t, "test_call", call.Name)
	assert.Len(t, call.Arguments, 2)
	assert.Equal(t, "value1", call.Arguments["param1"])
}

// TestMCPServer_HandleToolCall_EmptyArgs 测试处理空参数的工具调用
func TestMCPServer_HandleToolCall_EmptyArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	// 测试各种工具带空参数
	tools := []string{
		"check_login_status",
		"get_login_qrcode",
		"delete_cookies",
		"get_config",
		"get_stats",
	}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			call := MCPToolCall{
				Name:      toolName,
				Arguments: map[string]interface{}{},
			}
			result := server.HandleToolCall(ctx, call)
			require.NotNil(t, result)
		})
	}
}

// TestMCPServer_HandleToolCall_InvalidJobID 测试处理无效职位ID
func TestMCPServer_HandleToolCall_InvalidJobID(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "get_job_detail",
		Arguments: map[string]interface{}{
			"job_id": "", // 空ID
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}

// TestMCPServer_HandleToolCall_InvalidBatchDeliver 测试处理无效批量投递
func TestMCPServer_HandleToolCall_InvalidBatchDeliver(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	server := NewMCPServer(NewZhipinService())
	ctx := context.Background()

	call := MCPToolCall{
		Name: "batch_deliver",
		Arguments: map[string]interface{}{
			"job_ids": "not-an-array", // 错误的类型
		},
	}

	result := server.HandleToolCall(ctx, call)

	require.NotNil(t, result)
}
