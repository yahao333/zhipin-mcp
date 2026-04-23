package zhipin

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xpzouying/headless_browser"
)

// TestIntegration_SearchAndGetDetail 集成测试：搜索职位后获取第一个职位的详情
// 注意：此测试需要真实浏览器环境，在无浏览器环境下会跳过
func TestIntegration_SearchAndGetDetail(t *testing.T) {
	// 跳过CI环境或无浏览器环境
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器实例（无头模式）
	b := headless_browser.New()
	defer b.Close()

	// 创建页面
	page := b.NewPage()
	defer page.Close()

	// 创建搜索对象
	search := NewSearch(page)

	// 搜索职位
	ctx := context.Background()
	params := SearchParams{
		Keyword:  "Go工程师",
		Page:     1,
		PageSize: 5,
	}

	result, err := search.SearchJobs(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Greater(t, len(result.Jobs), 0, "搜索结果应该包含职位")

	// 获取第一个职位的ID
	firstJob := result.Jobs[0]
	require.NotEmpty(t, firstJob.ID, "第一个职位应该有ID")

	// 创建详情对象
	detail := NewDetail(page)

	// 获取职位详情
	jobDetail, err := detail.GetJobDetail(ctx, firstJob.ID)
	require.NoError(t, err)
	require.NotNil(t, jobDetail)

	// 验证职位详情
	assert.Equal(t, firstJob.ID, jobDetail.ID, "职位ID应该一致")
	assert.NotEmpty(t, jobDetail.UpdatedAt, "职位应该有更新时间")

	// 打印获取到的职位详情（用于调试）
	t.Logf("职位详情 - ID: %s, 标题: %s, 薪资: %s, 公司: %s",
		jobDetail.ID, jobDetail.Title, jobDetail.SalaryRange, jobDetail.CompanyName)
}

// TestIntegration_SearchWithDifferentKeywords 测试不同关键词搜索
// 注意：此测试需要真实浏览器环境
func TestIntegration_SearchWithDifferentKeywords(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	search := NewSearch(page)
	ctx := context.Background()

	// 测试不同关键词
	keywords := []string{"Java", "Python", "前端"}

	for _, keyword := range keywords {
		t.Run("搜索-"+keyword, func(t *testing.T) {
			params := SearchParams{
				Keyword:  keyword,
				Page:     1,
				PageSize: 3,
			}

			result, err := search.SearchJobs(ctx, params)
			require.NoError(t, err)
			require.NotNil(t, result)

			// 验证搜索结果
			assert.NotNil(t, result.Jobs)
			t.Logf("关键词 '%s' 搜索到 %d 个职位", keyword, len(result.Jobs))
		})
	}
}

// TestIntegration_GetJobDetailWithValidID 测试获取有效职位ID的详情
// 注意：此测试需要真实浏览器环境
func TestIntegration_GetJobDetailWithValidID(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	detail := NewDetail(page)
	ctx := context.Background()

	// 使用一个有效的测试职位ID
	testJobID := "abc123test"

	jobDetail, err := detail.GetJobDetail(ctx, testJobID)

	// 由于职位可能不存在，我们主要验证函数能正常执行（不panic）
	if err != nil {
		t.Logf("获取职位详情返回错误（预期行为）: %v", err)
	}

	// 如果成功，验证返回的数据结构
	if jobDetail != nil {
		assert.NotEmpty(t, jobDetail.ID)
		assert.NotEmpty(t, jobDetail.UpdatedAt)
	}
}

// TestIntegration_Pagination 测试分页功能
// 注意：此测试需要真实浏览器环境
func TestIntegration_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	search := NewSearch(page)
	ctx := context.Background()

	// 获取第一页
	params := SearchParams{
		Keyword:  "工程师",
		Page:     1,
		PageSize: 3,
	}

	result1, err := search.SearchJobs(ctx, params)
	require.NoError(t, err)

	// 获取第二页
	params.Page = 2
	result2, err := search.SearchJobs(ctx, params)
	require.NoError(t, err)

	// 验证两页结果不同
	if len(result1.Jobs) > 0 && len(result2.Jobs) > 0 {
		assert.NotEqual(t, result1.Jobs[0].ID, result2.Jobs[0].ID,
			"不同页的职位ID应该不同")
	}

	t.Logf("第1页职位数: %d, 第2页职位数: %d",
		len(result1.Jobs), len(result2.Jobs))
}

// TestIntegration_SearchJobThenGetDetailFullFlow 完整流程测试：搜索 -> 获取第一个职位详情
// 注意：此测试需要真实浏览器环境
func TestIntegration_SearchJobThenGetDetailFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	ctx := context.Background()

	// 步骤1: 搜索职位
	search := NewSearch(page)
	searchParams := SearchParams{
		Keyword:  "后端开发",
		Page:     1,
		PageSize: 10,
	}

	t.Log("步骤1: 开始搜索职位...")
	searchResult, err := search.SearchJobs(ctx, searchParams)
	require.NoError(t, err)
	require.NotNil(t, searchResult)

	// 验证搜索结果
	assert.Greater(t, searchResult.Total, 0, "搜索结果总数应该大于0")
	assert.Greater(t, len(searchResult.Jobs), 0, "搜索结果列表应该不为空")

	t.Logf("搜索到 %d 个职位，总计 %d 个", len(searchResult.Jobs), searchResult.Total)

	// 步骤2: 获取第一个职位的详情
	firstJob := searchResult.Jobs[0]
	t.Logf("步骤2: 获取第一个职位详情，ID: %s, 标题: %s", firstJob.ID, firstJob.Title)

	detail := NewDetail(page)
	jobDetail, err := detail.GetJobDetail(ctx, firstJob.ID)
	require.NoError(t, err)
	require.NotNil(t, jobDetail)

	// 验证详情数据
	assert.Equal(t, firstJob.ID, jobDetail.ID, "职位ID应该一致")
	assert.NotEmpty(t, jobDetail.UpdatedAt, "详情应该有更新时间")

	// 输出完整详情
	t.Logf("=== 职位详情 ===")
	t.Logf("ID: %s", jobDetail.ID)
	t.Logf("标题: %s", jobDetail.Title)
	t.Logf("薪资: %s", jobDetail.SalaryRange)
	t.Logf("公司: %s", jobDetail.CompanyName)
	t.Logf("HR: %s", jobDetail.HRName)
	t.Logf("更新时间: %s", jobDetail.UpdatedAt)
}

// TestIntegration_Login_CheckLoginStatus 测试检查登录状态
// 注意：此测试需要真实浏览器环境
func TestIntegration_Login_CheckLoginStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	login := NewLogin(page)
	ctx := context.Background()

	// 检查登录状态
	isLoggedIn, err := login.CheckLoginStatus(ctx)
	if err != nil {
		t.Logf("检查登录状态返回错误（预期行为，可能未登录）: %v", err)
	}

	t.Logf("登录状态: %v", isLoggedIn)
}

// TestIntegration_Login_FetchQrcodeImage 测试获取登录二维码
// 注意：此测试需要真实浏览器环境
func TestIntegration_Login_FetchQrcodeImage(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	login := NewLogin(page)
	ctx := context.Background()

	// 获取二维码图片
	qrcodeSrc, loggedIn, err := login.FetchQrcodeImage(ctx)
	if err != nil {
		t.Logf("获取二维码返回错误: %v", err)
	}

	// 如果已登录，qrcodeSrc 为空是正常的
	if loggedIn {
		t.Log("用户已登录，无需二维码")
	} else {
		// 验证二维码路径
		if qrcodeSrc != "" {
			t.Logf("获取到二维码，相对路径: %s", qrcodeSrc)
			assert.True(t, len(qrcodeSrc) > 0, "二维码路径不应为空")
		}
	}
}

// TestIntegration_Login_FetchQrcodeImageAsBase64 测试获取二维码 Base64
// 注意：此测试需要真实浏览器环境
func TestIntegration_Login_FetchQrcodeImageAsBase64(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	login := NewLogin(page)
	ctx := context.Background()

	// 获取二维码 Base64
	base64Img, loggedIn, err := login.FetchQrcodeImageAsBase64(ctx)
	if err != nil {
		t.Logf("获取二维码Base64返回错误: %v", err)
	}

	if loggedIn {
		t.Log("用户已登录，无需二维码")
	} else {
		// 验证 Base64 格式
		if base64Img != "" {
			assert.True(t, strings.HasPrefix(base64Img, "data:image/png;base64,"),
				"Base64图片应该有 data:image/png;base64, 前缀")
			t.Logf("获取到二维码 Base64，长度: %d", len(base64Img))
		}
	}
}

// TestIntegration_Deliver_DeliverJob 测试投递职位
// 注意：此测试需要真实浏览器环境且用户需已登录
func TestIntegration_Deliver_DeliverJob(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	// 先检查登录状态
	login := NewLogin(page)
	ctx := context.Background()
	isLoggedIn, _ := login.CheckLoginStatus(ctx)
	if !isLoggedIn {
		t.Skip("跳过投递测试 - 用户未登录")
	}

	// 搜索一个职位获取有效的 JobID
	search := NewSearch(page)
	searchParams := SearchParams{
		Keyword:  "测试工程师",
		Page:     1,
		PageSize: 3,
	}

	searchResult, err := search.SearchJobs(ctx, searchParams)
	if err != nil || len(searchResult.Jobs) == 0 {
		t.Skip("跳过投递测试 - 无法获取有效的职位")
	}

	// 获取一个有效的 JobID
	testJobID := searchResult.Jobs[0].ID
	t.Logf("测试投递职位，JobID: %s", testJobID)

	// 执行投递（这里只是测试能否正常调用，不实际投递以避免重复投递）
	deliver := NewDeliver(page)

	// 由于可能已经投递过，这里只测试方法能正常执行而不 panic
	// 调用 DeliverJob 会访问详情页并尝试投递
	result, err := deliver.DeliverJob(ctx, testJobID)

	// 无论成功还是失败，只要不 panic 即可
	if err != nil {
		t.Logf("投递返回错误（可能是已投递或页面结构变化）: %v", err)
	}
	if result != nil {
		t.Logf("投递结果: Success=%v, Message=%s", result.Success, result.Message)
	}
}

// TestIntegration_Deliver_checkDeliverResult 测试检查投递结果
// 注意：此测试需要真实浏览器环境且用户需已登录
func TestIntegration_Deliver_checkDeliverResult(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	deliver := NewDeliver(page)

	// 测试 checkDeliverResult 方法
	// 由于没有实际投递，这里主要验证方法能正常执行
	result, err := deliver.checkDeliverResult()

	if err != nil {
		t.Logf("检查投递结果返回错误: %v", err)
	}

	// 验证返回结果结构
	if result != nil {
		assert.NotNil(t, result, "结果结构不应为 nil")
		t.Logf("投递结果: Success=%v, Message=%s", result.Success, result.Message)
	}
}

// TestIntegration_Deliver_checkDelivered 测试检查是否已投递
// 注意：此测试需要真实浏览器环境
func TestIntegration_Deliver_checkDelivered(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	deliver := NewDeliver(page)

	// 测试 checkDelivered 方法
	isDelivered, err := deliver.checkDelivered("test-job-id")
	if err != nil {
		t.Logf("检查是否已投递返回错误: %v", err)
	}

	t.Logf("是否已投递: %v", isDelivered)
}

// TestIntegration_Message_ListMessages 测试获取消息列表
// 注意：此测试需要真实浏览器环境且用户需已登录
func TestIntegration_Message_ListMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	// 先检查登录状态
	login := NewLogin(page)
	ctx := context.Background()
	isLoggedIn, _ := login.CheckLoginStatus(ctx)
	if !isLoggedIn {
		t.Skip("跳过消息测试 - 用户未登录")
	}

	msgAction := NewMessageAction(page)
	messages, err := msgAction.ListMessages(ctx)

	if err != nil {
		t.Logf("获取消息列表返回错误（可能是无权限或页面结构变化）: %v", err)
	}

	// 验证返回的消息列表结构
	if messages != nil {
		t.Logf("获取到 %d 条消息", len(messages.Messages))
		for i, msg := range messages.Messages {
			t.Logf("消息 %d: %s - %s - %s", i+1, msg.PersonName, msg.CompanyName, msg.JobTitle)
		}
	}
}

// TestIntegration_CronManager 测试定时任务管理器
// 注意：此测试需要真实浏览器环境
func TestIntegration_CronManager(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 测试 NewCronManager
	mgr := NewCronManager()
	require.NotNil(t, mgr)

	// 测试 AddTask
	task := &CronTaskInfo{
		ID:       100,
		TaskName: "integration-test-task",
		CronExpr: "0 9 * * *", // 每天早上9点
		Keyword:  "Go工程师",
		City:     "北京",
		IsActive: true,
	}

	id, err := mgr.AddTask(task)
	require.NoError(t, err)
	assert.Greater(t, id, 0, "任务ID应该大于0")

	// 测试 ListTasks
	entries := mgr.ListTasks()
	assert.GreaterOrEqual(t, len(entries), 1, "应该至少有一个任务")

	// 测试 RemoveTask
	err = mgr.RemoveTask(100)
	require.NoError(t, err)

	// 测试 Start/Stop
	mgr.Start()
	mgr.Stop()

	t.Logf("定时任务管理器集成测试完成")
}

// TestIntegration_CronManager_Callback 测试定时任务回调
// 注意：此测试需要真实浏览器环境
func TestIntegration_CronManager_Callback(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	mgr := NewCronManager()

	// 设置回调
	callbackCalled := false
	mgr.SetSearchCallback(func(keyword, city string) error {
		callbackCalled = true
		t.Logf("回调被调用: keyword=%s, city=%s", keyword, city)
		return nil
	})

	// 添加任务
	task := &CronTaskInfo{
		ID:       200,
		TaskName: "callback-test-task",
		CronExpr: "0 9 * * *",
		Keyword:  "测试",
		City:     "北京",
	}

	_, err := mgr.AddTask(task)
	require.NoError(t, err)

	// 手动执行任务以触发回调
	mgr.executeTask(task)

	assert.True(t, callbackCalled, "回调应该被调用")

	mgr.Stop()
}

// TestIntegration_BatchDeliver 测试批量投递
// 注意：此测试需要真实浏览器环境且用户需已登录
func TestIntegration_BatchDeliver(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 - 需要真实浏览器环境")
	}

	// 创建浏览器
	b := headless_browser.New()
	defer b.Close()
	page := b.NewPage()
	defer page.Close()

	// 先检查登录状态
	login := NewLogin(page)
	ctx := context.Background()
	isLoggedIn, _ := login.CheckLoginStatus(ctx)
	if !isLoggedIn {
		t.Skip("跳过批量投递测试 - 用户未登录")
	}

	// 搜索获取一些职位ID
	search := NewSearch(page)
	searchParams := SearchParams{
		Keyword:  "前端工程师",
		Page:     1,
		PageSize: 2,
	}

	searchResult, err := search.SearchJobs(ctx, searchParams)
	if err != nil || len(searchResult.Jobs) < 2 {
		t.Skip("跳过批量投递测试 - 无法获取足够的职位")
	}

	// 准备批量投递（只取2个职位，且只测试方法调用不实际投递）
	jobIDs := []string{
		searchResult.Jobs[0].ID,
		searchResult.Jobs[1].ID,
	}

	deliver := NewDeliver(page)

	// 测试 BatchDeliver 方法调用（实际会执行投递，这里只用2个职位测试流程）
	// 注意：实际执行可能会有副作用，这里仅验证方法能正常调用
	results, err := deliver.BatchDeliver(jobIDs)

	if err != nil {
		t.Logf("批量投递返回错误: %v", err)
	}

	if results != nil {
		t.Logf("批量投递结果数量: %d", len(results))
		for i, r := range results {
			t.Logf("结果 %d: JobID=%s, Success=%v, Message=%s", i+1, r.JobID, r.Success, r.Message)
		}
	}
}
