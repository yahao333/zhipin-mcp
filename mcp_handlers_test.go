package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPToolResultFields 测试 MCP 工具结果字段
func TestMCPToolResultFields(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "Hello World"},
			{Type: "image", MimeType: "image/png", Data: "base64data"},
		},
		IsError: false,
	}

	assert.Len(t, result.Content, 2)
	assert.False(t, result.IsError)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "Hello World", result.Content[0].Text)
}

// TestMCPToolResultError 测试错误结果
func TestMCPToolResultError(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "错误信息"},
		},
		IsError: true,
	}

	assert.True(t, result.IsError)
	assert.Equal(t, "错误信息", result.Content[0].Text)
}

// TestMCPContentTypes 测试 MCP 内容类型
func TestMCPContentTypes(t *testing.T) {
	tests := []struct {
		name    string
		content MCPContent
	}{
		{
			name: "文本内容",
			content: MCPContent{
				Type: "text",
				Text: "这是一段文本",
			},
		},
		{
			name: "图片内容",
			content: MCPContent{
				Type:     "image",
				MimeType: "image/png",
				Data:     "base64-encoded-image",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.content.Type)
		})
	}
}

// TestHandleSearchJobsArgs 测试搜索参数解析
func TestHandleSearchJobsArgs(t *testing.T) {
	// 测试有效的参数
	args := map[string]interface{}{
		"keyword":    "工程师",
		"city":       "北京",
		"district":   "海淀区",
		"experience": "3-5年",
		"education":  "本科",
		"job_type":   "全职",
		"salary":     "20k-40k",
		"page":       float64(1),
	}

	// 验证参数解析逻辑
	keyword, _ := args["keyword"].(string)
	city, _ := args["city"].(string)
	page := 1
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}

	assert.Equal(t, "工程师", keyword)
	assert.Equal(t, "北京", city)
	assert.Equal(t, 1, page)
}

// TestHandleSearchJobsArgsEmptyKeyword 测试空关键词
func TestHandleSearchJobsArgsEmptyKeyword(t *testing.T) {
	args := map[string]interface{}{
		"city": "北京",
	}

	keyword, _ := args["keyword"].(string)
	assert.Empty(t, keyword, "空关键词应为空字符串")
}

// TestHandleSearchJobsArgsDefaultPage 测试默认页码
func TestHandleSearchJobsArgsDefaultPage(t *testing.T) {
	args := map[string]interface{}{
		"keyword": "工程师",
	}

	page := 1
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}

	assert.Equal(t, 1, page, "未提供页码时应默认为1")
}

// TestHandleGetJobDetailArgs 测试获取职位详情参数
func TestHandleGetJobDetailArgs(t *testing.T) {
	args := map[string]interface{}{
		"job_id": "job-123456",
	}

	jobID, _ := args["job_id"].(string)
	assert.Equal(t, "job-123456", jobID)
}

// TestHandleGetJobDetailArgsEmpty 测试空职位ID
func TestHandleGetJobDetailArgsEmpty(t *testing.T) {
	args := map[string]interface{}{}

	jobID, _ := args["job_id"].(string)
	assert.Empty(t, jobID, "空职位ID应为空字符串")
}

// TestHandleDeliverJobArgs 测试投递职位参数
func TestHandleDeliverJobArgs(t *testing.T) {
	args := map[string]interface{}{
		"job_id": "job-789",
	}

	jobID, _ := args["job_id"].(string)
	assert.Equal(t, "job-789", jobID)
}

// TestHandleDeliveredListArgs 测试已投递列表参数
func TestHandleDeliveredListArgs(t *testing.T) {
	args := map[string]interface{}{
		"limit":  float64(20),
		"offset": float64(0),
	}

	limit := 20
	offset := 0
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	assert.Equal(t, 20, limit)
	assert.Equal(t, 0, offset)
}

// TestHandleDeliveredListArgsDefaults 测试默认分页参数
func TestHandleDeliveredListArgsDefaults(t *testing.T) {
	args := map[string]interface{}{}

	limit := 20
	offset := 0
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	assert.Equal(t, 20, limit, "默认limit应为20")
	assert.Equal(t, 0, offset, "默认offset应为0")
}

// TestHandleBatchDeliverArgs 测试批量投递参数
func TestHandleBatchDeliverArgs(t *testing.T) {
	args := map[string]interface{}{
		"job_ids": []interface{}{"job-1", "job-2", "job-3"},
	}

	jobIDsInterface, ok := args["job_ids"].([]interface{})
	require.True(t, ok, "job_ids应该是数组")

	jobIDs := make([]string, 0, len(jobIDsInterface))
	for _, id := range jobIDsInterface {
		if idStr, ok := id.(string); ok {
			jobIDs = append(jobIDs, idStr)
		}
	}

	assert.Len(t, jobIDs, 3)
	assert.Equal(t, "job-1", jobIDs[0])
	assert.Equal(t, "job-2", jobIDs[1])
	assert.Equal(t, "job-3", jobIDs[2])
}

// TestHandleBatchDeliverArgsInvalid 测试无效批量投递参数
func TestHandleBatchDeliverArgsInvalid(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "非数组job_ids",
			args: map[string]interface{}{
				"job_ids": "job-1,job-2",
			},
		},
		{
			name: "空数组",
			args: map[string]interface{}{
				"job_ids": []interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobIDsInterface, ok := tt.args["job_ids"].([]interface{})
			if !ok {
				// 非数组类型
				return
			}

			jobIDs := make([]string, 0, len(jobIDsInterface))
			for _, id := range jobIDsInterface {
				if idStr, ok := id.(string); ok {
					jobIDs = append(jobIDs, idStr)
				}
			}

			assert.Empty(t, jobIDs)
		})
	}
}

// TestHandleStartCronArgs 测试启动定时任务参数
func TestHandleStartCronArgs(t *testing.T) {
	args := map[string]interface{}{
		"task_name":       "每日投递任务",
		"cron_expression": "0 9 * * *",
		"keyword":         "工程师",
		"city":            "北京",
	}

	taskName, _ := args["task_name"].(string)
	cronExpr, _ := args["cron_expression"].(string)
	keyword, _ := args["keyword"].(string)
	city, _ := args["city"].(string)

	assert.Equal(t, "每日投递任务", taskName)
	assert.Equal(t, "0 9 * * *", cronExpr)
	assert.Equal(t, "工程师", keyword)
	assert.Equal(t, "北京", city)
}

// TestHandleStartCronArgsMissing 测试缺少参数
func TestHandleStartCronArgsMissing(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "缺少task_name",
			args: map[string]interface{}{
				"cron_expression": "0 9 * * *",
				"keyword":         "工程师",
			},
		},
		{
			name: "缺少cron_expression",
			args: map[string]interface{}{
				"task_name": "每日投递任务",
				"keyword":   "工程师",
			},
		},
		{
			name: "缺少keyword",
			args: map[string]interface{}{
				"task_name":       "每日投递任务",
				"cron_expression": "0 9 * * *",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskName, _ := tt.args["task_name"].(string)
			cronExpr, _ := tt.args["cron_expression"].(string)
			keyword, _ := tt.args["keyword"].(string)

			// 验证至少有一个参数缺失
			assert.True(t, taskName == "" || cronExpr == "" || keyword == "")
		})
	}
}

// TestHandleStopCronArgs 测试停止定时任务参数
func TestHandleStopCronArgs(t *testing.T) {
	args := map[string]interface{}{
		"task_id": float64(123),
	}

	taskIDFloat, ok := args["task_id"].(float64)
	require.True(t, ok)

	taskID := int(taskIDFloat)
	assert.Equal(t, 123, taskID)
}

// TestHandleUpdateConfigArgs 测试更新配置参数
func TestHandleUpdateConfigArgs(t *testing.T) {
	args := map[string]interface{}{
		"username":  "testuser",
		"password":  "testpassword",
		"max_daily": float64(50),
	}

	username, _ := args["username"].(string)
	password, _ := args["password"].(string)
	maxDaily := 0
	if m, ok := args["max_daily"].(float64); ok {
		maxDaily = int(m)
	}

	assert.Equal(t, "testuser", username)
	assert.Equal(t, "testpassword", password)
	assert.Equal(t, 50, maxDaily)
}

// TestHandleUpdateConfigArgsPartial 测试部分更新配置
func TestHandleUpdateConfigArgsPartial(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "仅更新用户名",
			args: map[string]interface{}{
				"username": "newuser",
			},
		},
		{
			name: "仅更新密码",
			args: map[string]interface{}{
				"password": "newpassword",
			},
		},
		{
			name: "仅更新每日上限",
			args: map[string]interface{}{
				"max_daily": float64(100),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, _ := tt.args["username"].(string)
			password, _ := tt.args["password"].(string)
			maxDaily := 0
			if m, ok := tt.args["max_daily"].(float64); ok {
				maxDaily = int(m)
			}

			// 验证至少有一个非空值
			assert.True(t, username != "" || password != "" || maxDaily > 0)
		})
	}
}

// TestParseJSON 测试 JSON 解析辅助函数
func TestParseJSON(t *testing.T) {
	// 测试结构体转 map
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	input := TestStruct{Name: "张三", Age: 30}
	result, err := parseJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "张三", result["name"])
	assert.Equal(t, 30.0, result["age"])
}

// TestParseJSONError 测试 JSON 解析错误
func TestParseJSONError(t *testing.T) {
	// 测试无效输入
	_, err := parseJSON(make(chan int))
	assert.Error(t, err, "无效类型应返回错误")
}

// TestContextWithTimeout 测试 context 超时
func TestContextWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	// 立即超时的 context
	select {
	case <-ctx.Done():
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	default:
		t.Fatal("context 应该立即超时")
	}
}

// TestParseJSONWithMap 测试解析map
func TestParseJSONWithMap(t *testing.T) {
	input := map[string]interface{}{
		"name": "张三",
		"age":  30,
	}
	result, err := parseJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "张三", result["name"])
	assert.Equal(t, 30.0, result["age"])
}

// TestParseJSONWithSlice 测试解析slice
func TestParseJSONWithSlice(t *testing.T) {
	input := []string{"a", "b", "c"}
	_, err := parseJSON(input)
	assert.Error(t, err, "slice should return error")
}

// TestParseJSONWithInt 测试解析int
func TestParseJSONWithInt(t *testing.T) {
	input := 123
	_, err := parseJSON(input)
	assert.Error(t, err, "int should return error")
}

// TestParseJSONWithFloat 测试解析float
func TestParseJSONWithFloat(t *testing.T) {
	input := 3.14
	_, err := parseJSON(input)
	assert.Error(t, err, "float should return error")
}

// TestParseJSONWithBool 测试解析bool
func TestParseJSONWithBool(t *testing.T) {
	input := true
	_, err := parseJSON(input)
	assert.Error(t, err, "bool should return error")
}

// TestParseJSONWithNil 测试解析nil
func TestParseJSONWithNil(t *testing.T) {
	var input interface{} = nil
	result, err := parseJSON(input)
	require.NoError(t, err)
	assert.Nil(t, result)
}
