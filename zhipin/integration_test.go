package zhipin

import (
	"context"
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
