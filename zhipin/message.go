package zhipin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/pkg/debug"
)

// MessageStatus 消息状态
type MessageStatus string

const (
	MessageStatusDelivered MessageStatus = "delivered" // 已送达
	MessageStatusRead      MessageStatus = "read"      // 已读
	MessageStatusUnknown   MessageStatus = "unknown"   // 未知
)

// Message 消息结构
type Message struct {
	PersonName    string        // 人名称（HR姓名）
	CompanyName   string        // 公司名称
	JobTitle      string        // 职位名称
	Avatar        string        // 头像URL
	MessageDigest string        // 消息摘要（最新一条消息内容）
	Time          time.Time     // 最新消息时间
	UnreadCount   int           // 未读消息数量
	Status        MessageStatus // 消息状态（已送达/已读/未知）
}

// MessageList 消息列表
type MessageList struct {
	Messages []Message
}

// MessageAction 消息操作
type MessageAction struct {
	page *rod.Page
}

// NewMessageAction 创建消息操作实例
func NewMessageAction(page *rod.Page) *MessageAction {
	return &MessageAction{page: page}
}

// ListMessages 获取消息列表
func (m *MessageAction) ListMessages(ctx context.Context) (*MessageList, error) {
	logrus.Debugf("[MessageAction.ListMessages] ========== 开始获取消息列表 ==========")

	// 导航到消息页面
	url := "https://www.zhipin.com/web/geek/chat"
	logrus.Debugf("[MessageAction.ListMessages] 准备导航到 URL: %s", url)

	if err := m.page.Navigate(url); err != nil {
		logrus.Errorf("[MessageAction.ListMessages] Navigate 失败: %v", err)
		return nil, err
	}
	logrus.Debugf("[MessageAction.ListMessages] Navigate 成功")

	// 等待页面加载
	logrus.Debugf("[MessageAction.ListMessages] 等待页面加载...")
	m.page.WaitLoad()
	logrus.Debugf("[MessageAction.ListMessages] 页面加载完成")

	// 等待一下让动态内容加载
	time.Sleep(2 * time.Second)

	// 解析消息列表
	logrus.Debugf("[MessageAction.ListMessages] 开始解析消息列表...")

	messages, err := m.parseMessageList()
	if err != nil {
		logrus.Errorf("[MessageAction.ListMessages] parseMessageList 失败: %v", err)
		return nil, err
	}

	logrus.Debugf("[MessageAction.ListMessages] 解析完成, 共 %d 条消息", len(messages))
	logrus.Debugf("[MessageAction.ListMessages] ========== 获取消息列表完成 ==========")

	return &MessageList{Messages: messages}, nil
}

// parseMessageList 解析消息列表
func (m *MessageAction) parseMessageList() ([]Message, error) {
	logrus.Debugf("[MessageAction.parseMessageList] ========== 开始解析消息列表 ==========")

	var messages []Message

	// 尝试多个选择器找到消息列表容器
	listSelectors := []string{
		".friend-item",                   // BOSS直聘的消息项类名
		"[role='listitem']",              // role="listitem" 的 li 元素
		"ul[class*='conversation'] > li", // conversation 容器的直接 li 子元素
		".chat-item",
		".dialog-item",
		".message-item",
		".conversation-item",
		"[class*='chat-item']",
		"[class*='dialog-item']",
		"[class*='message-item']",
		"[class*='conversation']",
	}

	var items rod.Elements
	var err error

	for _, selector := range listSelectors {
		logrus.Debugf("[MessageAction.parseMessageList] 尝试选择器: %s", selector)
		items, err = m.page.Elements(selector)
		if err == nil && len(items) > 0 {
			logrus.Debugf("[MessageAction.parseMessageList] 使用选择器 %s 找到 %d 个元素", selector, len(items))
			break
		}
	}

	if len(items) == 0 {
		logrus.Warnf("[MessageAction.parseMessageList] 未找到消息列表元素，尝试获取页面HTML进行调试")
		html, _ := m.page.HTML()
		logrus.Debugf("[MessageAction.parseMessageList] 页面HTML长度: %d", len(html))
		return messages, nil
	}

	// 解析每个消息项
	for i, item := range items {
		logrus.Debugf("[MessageAction.parseMessageList] 解析第 %d/%d 个消息项", i+1, len(items))

		// 调试：打印当前消息项的 HTML 结构（如果有 figure 的话）
		if figEl, err := item.Element(".figure"); err == nil {
			if html, err := figEl.HTML(); err == nil {
				if len(html) > 500 {
					html = html[:500] + "..."
				}
				logrus.Debugf("[MessageAction.parseMessageList] .figure HTML: %s", html)
			}
		} else {
			logrus.Debugf("[MessageAction.parseMessageList] 未找到 .figure 元素, err=%v", err)
			// 打印当前元素的 class 属性
			if classAttr, err := item.Attribute("class"); err == nil && classAttr != nil {
				logrus.Debugf("[MessageAction.parseMessageList] 当前元素 class=%s", *classAttr)
			}
		}

		msg := Message{}

		// 解析人名称（HR姓名）
		nameSelectors := []string{
			".title-box .name-text", // BOSS直聘：title-box 下的 name-text
			".title-box .name-box",  // 备选：完整 name-box
			".name-text",
			".name",
			".person-name",
			".chat-name",
			"[class*='name-text']",
			"[class*='name-box']",
			".nick-name",
		}
		for _, sel := range nameSelectors {
			el, err := item.Element(sel)
			if err == nil {
				msg.PersonName, _ = el.Text()
				msg.PersonName = strings.TrimSpace(msg.PersonName)
				if msg.PersonName != "" {
					break
				}
			}
		}

		// 解析公司名称
		companySelectors := []string{
			".title-box .name-box > span:nth-child(2)", // BOSS直聘：name-box 下的第2个span
			".title-box .name-box span:nth-child(2)",
			".company-name",
			"[class*='company']",
			".sub",
		}
		for _, sel := range companySelectors {
			el, err := item.Element(sel)
			if err == nil {
				msg.CompanyName, _ = el.Text()
				msg.CompanyName = strings.TrimSpace(msg.CompanyName)
				if msg.CompanyName != "" {
					break
				}
			}
		}

		// 解析职位名称
		jobSelectors := []string{
			".title-box .name-box > span:nth-child(4)", // BOSS直聘：name-box 下的第4个span（职位/HRBP）
			".title-box .name-box span:nth-child(4)",
			".job-title",
			"[class*='job']",
			".position",
		}
		for _, sel := range jobSelectors {
			el, err := item.Element(sel)
			if err == nil {
				msg.JobTitle, _ = el.Text()
				msg.JobTitle = strings.TrimSpace(msg.JobTitle)
				if msg.JobTitle != "" {
					break
				}
			}
		}

		// 解析消息摘要
		digestSelectors := []string{
			".last-msg-text", // BOSS直聘：消息摘要
			".last-message",
			".digest",
			".message-digest",
			"[class*='digest']",
			"[class*='last-msg']",
			"[class*='message']",
		}
		for _, sel := range digestSelectors {
			el, err := item.Element(sel)
			if err == nil {
				msg.MessageDigest, _ = el.Text()
				msg.MessageDigest = strings.TrimSpace(msg.MessageDigest)
				if msg.MessageDigest != "" {
					break
				}
			}
		}

		// 解析时间
		timeSelectors := []string{
			".time",
			"[class*='time']",
			".date",
		}
		for _, sel := range timeSelectors {
			el, err := item.Element(sel)
			if err == nil {
				timeStr, _ := el.Text()
				timeStr = strings.TrimSpace(timeStr)
				if timeStr != "" {
					msg.Time = parseRelativeTime(timeStr)
					break
				}
			}
		}

		// 解析未读数量 - 注意 notice-badge 在 figure 元素内部
		unreadSelectors := []string{
			".figure .notice-badge",
			"span.notice-badge",
			"[class*='notice-badge']",
			".unread",
			".badge",
		}
		logrus.Debugf("[MessageAction.parseMessageList] (unread) 开始解析未读数量，候选选择器数量=%d", len(unreadSelectors))
		originalUnread := msg.UnreadCount
		for _, sel := range unreadSelectors {
			logrus.Debugf("[MessageAction.parseMessageList] (unread) 尝试选择器: %s", sel)
			el, err := item.Element(sel)
			if err != nil {
				logrus.Debugf("[MessageAction.parseMessageList] (unread) 选择器未命中: %s, err=%v", sel, err)
				continue
			}

			unreadRaw, textErr := el.Text()
			unreadStr := strings.TrimSpace(unreadRaw)

			classAttr, _ := el.Attribute("class")
			classVal := ""
			if classAttr != nil {
				classVal = *classAttr
			}

			if html, htmlErr := el.HTML(); htmlErr == nil {
				htmlSnippet := html
				if len(htmlSnippet) > 200 {
					htmlSnippet = htmlSnippet[:200] + "..."
				}
				logrus.Debugf("[MessageAction.parseMessageList] (unread) 命中选择器=%s, rawText=%q, trimText=%q, class=%q, htmlLen=%d, htmlSnippet=%q, textErr=%v",
					sel, unreadRaw, unreadStr, classVal, len(html), htmlSnippet, textErr)
			} else {
				logrus.Debugf("[MessageAction.parseMessageList] (unread) 命中选择器=%s, rawText=%q, trimText=%q, class=%q, htmlErr=%v, textErr=%v",
					sel, unreadRaw, unreadStr, classVal, htmlErr, textErr)
			}

			if unreadStr != "" && unreadStr != "0" {
				count, parseErr := parseInt(unreadStr)
				logrus.Debugf("[MessageAction.parseMessageList] (unread) 尝试解析文本为数字: text=%q, count=%d, parseErr=%v", unreadStr, count, parseErr)
				if count > 0 {
					msg.UnreadCount = count
				}
			} else if unreadStr == "" {
				hasClass, _, hasErr := el.Has("span.notice-badge")
				logrus.Debugf("[MessageAction.parseMessageList] (unread) 文本为空，尝试通过结构判断是否未读: hasSpanNoticeBadge=%v, hasErr=%v, currentUnread=%d",
					hasClass, hasErr, msg.UnreadCount)
				if hasClass {
					msg.UnreadCount = 1
				}
			} else {
				logrus.Debugf("[MessageAction.parseMessageList] (unread) 文本为 0，视为无未读: text=%q", unreadStr)
			}

			logrus.Debugf("[MessageAction.parseMessageList] (unread) 选择器=%s 解析完成: originalUnread=%d -> currentUnread=%d", sel, originalUnread, msg.UnreadCount)
			break
		}
		logrus.Debugf("[MessageAction.parseMessageList] (unread) 解析未读数量结束: finalUnread=%d", msg.UnreadCount)

		// 判断消息状态
		if msg.UnreadCount > 0 {
			msg.Status = MessageStatusDelivered
		} else {
			msg.Status = MessageStatusRead
		}

		// 解析头像 - 使用 figure img 选择器
		avatarSelectors := []string{
			".figure img",
			"img[src*='avatar']",
			".avatar img",
			"[class*='avatar'] img",
		}
		for _, sel := range avatarSelectors {
			el, err := item.Element(sel)
			if err == nil {
				src, _ := el.Attribute("src")
				if src != nil {
					msg.Avatar = *src
					break
				}
			}
		}

		logrus.Debugf("[MessageAction.parseMessageList] 解析结果: PersonName=%s, Company=%s, Job=%s, Digest=%s",
			msg.PersonName, msg.CompanyName, msg.JobTitle, msg.MessageDigest)

		// 只添加有有效人名称的消息
		if msg.PersonName != "" {
			messages = append(messages, msg)
		}
	}

	logrus.Debugf("[MessageAction.parseMessageList] ========== 解析消息列表完成, 共 %d 条 ==========", len(messages))
	return messages, nil
}

// parseRelativeTime 解析相对时间字符串
func parseRelativeTime(timeStr string) time.Time {
	now := time.Now()

	// 尝试解析 "11:52" 格式（今天的时间）
	if strings.Contains(timeStr, ":") {
		if t, err := time.ParseInLocation("15:04", timeStr, time.Local); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		}
	}

	// 尝试解析 "03月27日" 格式
	if strings.Contains(timeStr, "月") && strings.Contains(timeStr, "日") {
		// 替换月日为标准格式
		formatted := strings.ReplaceAll(timeStr, "月", "-")
		formatted = strings.ReplaceAll(formatted, "日", "")
		if t, err := time.ParseInLocation("2006-01-02", formatted, time.Local); err == nil {
			// 如果年份解析为0，使用当前年份
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
			}
			return t
		}
	}

	// 尝试解析常见格式
	// 今天
	if strings.Contains(timeStr, "今天") || strings.Contains(timeStr, "今日") {
		return now
	}

	// 昨天
	if strings.Contains(timeStr, "昨天") || strings.Contains(timeStr, "昨日") {
		return now.AddDate(0, 0, -1)
	}

	// 前天
	if strings.Contains(timeStr, "前天") {
		return now.AddDate(0, 0, -2)
	}

	// 几天前
	if strings.Contains(timeStr, "天前") {
		var days int
		if _, err := parseInt(timeStr); err == nil {
			days, _ = parseInt(timeStr)
			return now.AddDate(0, 0, -days)
		}
	}

	// 几小时前
	if strings.Contains(timeStr, "小时前") || strings.Contains(timeStr, "h前") {
		var hours int
		if _, err := parseInt(timeStr); err == nil {
			hours, _ = parseInt(timeStr)
			return now.Add(time.Duration(-hours) * time.Hour)
		}
	}

	// 几分钟前
	if strings.Contains(timeStr, "分钟前") || strings.Contains(timeStr, "min前") {
		var mins int
		if _, err := parseInt(timeStr); err == nil {
			mins, _ = parseInt(timeStr)
			return now.Add(time.Duration(-mins) * time.Minute)
		}
	}

	// 刚刚
	if strings.Contains(timeStr, "刚刚") || strings.Contains(timeStr, "just") {
		return now
	}

	return now
}

// parseInt 解析字符串中的数字
func parseInt(s string) (int, error) {
	var num int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num, nil
}

// DeleteMessage 删除消息
// 通过 person_name, company_name, job_title 匹配消息，然后点击删除按钮
func (m *MessageAction) DeleteMessage(ctx context.Context, personName, companyName, jobTitle string) error {
	logrus.Debugf("[MessageAction.DeleteMessage] ========== 开始删除消息 ==========")
	logrus.Debugf("[MessageAction.DeleteMessage] 筛选条件: personName=%s, companyName=%s, jobTitle=%s", personName, companyName, jobTitle)

	// 步骤1: 获取消息列表
	messages, err := m.ListMessages(ctx)
	if err != nil {
		logrus.Errorf("[MessageAction.DeleteMessage] 获取消息列表失败: %v", err)
		return err
	}
	logrus.Debugf("[MessageAction.DeleteMessage] 获取到 %d 条消息", len(messages.Messages))

	// 步骤2: 查找匹配的消息
	var targetItem rod.Elements
	var found bool

	for _, selector := range []string{".friend-item", "[role='listitem']", ".chat-item", ".dialog-item", ".message-item"} {
		items, err := m.page.Elements(selector)
		if err != nil || len(items) == 0 {
			continue
		}

		for i, item := range items {
			// 获取当前项的文本内容进行匹配
			itemText, _ := item.Text()
			logrus.Debugf("[MessageAction.DeleteMessage] 检查第 %d 个元素, 文本长度: %d", i, len(itemText))

			// 查找匹配的消息
			var nameEl interface{ Text() (string, error) }
			for _, sel := range []string{".title-box .name-text", ".name-text", ".name"} {
				if el, err := item.Element(sel); err == nil {
					nameEl = el
					break
				}
			}
			if nameEl == nil {
				continue
			}

			nameText, _ := nameEl.Text()
			nameText = strings.TrimSpace(nameText)
			logrus.Debugf("[MessageAction.DeleteMessage] 匹配人名: %s vs %s", nameText, personName)

			// 模糊匹配人名
			if personName != "" && !strings.Contains(nameText, personName) {
				continue
			}

			// 获取公司名称
			companyMatch := true
			if companyName != "" {
				var cEl interface{ Text() (string, error) }
				for _, sel := range []string{".title-box .name-box > span:nth-child(2)", ".company-name", "[class*='company']"} {
					if el, err := item.Element(sel); err == nil {
						cEl = el
						break
					}
				}
				if cEl != nil {
					cText, _ := cEl.Text()
					cText = strings.TrimSpace(cText)
					companyMatch = strings.Contains(cText, companyName)
					logrus.Debugf("[MessageAction.DeleteMessage] 匹配公司: %s vs %s, 匹配=%v", cText, companyName, companyMatch)
				}
			}

			// 获取职位名称
			jobMatch := true
			if jobTitle != "" {
				var jEl interface{ Text() (string, error) }
				for _, sel := range []string{".title-box .name-box > span:nth-child(4)", ".job-title", "[class*='job']"} {
					if el, err := item.Element(sel); err == nil {
						jEl = el
						break
					}
				}
				if jEl != nil {
					jText, _ := jEl.Text()
					jText = strings.TrimSpace(jText)
					jobMatch = strings.Contains(jText, jobTitle)
					logrus.Debugf("[MessageAction.DeleteMessage] 匹配职位: %s vs %s, 匹配=%v", jText, jobTitle, jobMatch)
				}
			}

			// 所有条件都匹配
			if companyMatch && jobMatch {
				targetItem = rod.Elements{item}
				found = true
				logrus.Infof("[MessageAction.DeleteMessage] 找到匹配的消息，人名: %s", nameText)
				break
			}
		}

		if found {
			break
		}
	}

	if !found {
		logrus.Warnf("[MessageAction.DeleteMessage] 未找到匹配的消息")
		return errors.New("未找到匹配的消息")
	}

	// 步骤3: 鼠标悬停到目标元素，显示操作按钮
	item := targetItem[0]
	// 打印 targetItem 的 HTML 信息用于调试
	if html, err := item.HTML(); err == nil {
		// if len(html) > 1000 {
		// 	html = html[:1000] + "..."
		// }
		logrus.Debugf("[MessageAction.DeleteMessage] targetItem HTML: %s", html)
	} else {
		logrus.Debugf("[MessageAction.DeleteMessage] 获取 targetItem HTML 失败: %v", err)
	}
	logrus.Debugf("[MessageAction.DeleteMessage] 鼠标悬停到消息项")

	// 使用 MouseHover 方法替代 Hover
	err = item.Hover()
	if err != nil {
		logrus.Errorf("[MessageAction.DeleteMessage] Hover 失败: %v", err)
		return err
	}

	// 等待按钮出现
	time.Sleep(500 * time.Millisecond)

	// 步骤4: 查找并点击删除按钮
	logrus.Debugf("[MessageAction.DeleteMessage] 查找删除按钮")
	err = m.clickDeleteButton(item)
	if err != nil {
		logrus.Errorf("[MessageAction.DeleteMessage] 点击删除按钮失败: %v", err)
		return err
	}

	logrus.Infof("[MessageAction.DeleteMessage] ========== 删除消息完成 ==========")
	return nil
}

// moveMouseTo 使用 go-rod 的 Mouse.MoveToElement 方法移动鼠标到元素
func (m *MessageAction) moveMouseTo(el *rod.Element) error {
	// 调试：先打印元素信息
	elInfo, _ := el.HTML()
	if len(elInfo) > 200 {
		elInfo = elInfo[:200] + "..."
	}
	logrus.Debugf("[MessageAction.moveMouseTo] 元素HTML: %s", elInfo)

	// 使用 el.Eval() 获取元素的 getBoundingClientRect 信息用于调试
	result, err := el.Eval(`function() {
		var rect = this.getBoundingClientRect();
		return {
			left: rect.left,
			top: rect.top,
			width: rect.width,
			height: rect.height,
			x: rect.left + rect.width / 2,
			y: rect.top + rect.height / 2
		};
	}`)
	if err == nil {
		resultStr := result.Value.JSON("", "")
		logrus.Debugf("[MessageAction.moveMouseTo] getBoundingClientRect: %s", resultStr)
	}

	// 使用 go-rod 的 Mouse.MoveToElement 方法移动鼠标到元素中心
	// 这个方法会自动处理元素的定位
	logrus.Debugf("[MessageAction.moveMouseTo] 使用 Mouse.MoveToElement 移动鼠标到元素")
	err = el.Hover()
	if err != nil {
		logrus.Errorf("[MessageAction.moveMouseTo] Hover 失败: %v", err)
		return err
	}

	logrus.Debugf("[MessageAction.moveMouseTo] 鼠标移动完成")
	return nil
}

// clickDeleteButton 在消息项中点击删除按钮
// 交互流程：
// 1. Hover 到消息项 → 显示灰色的三个点图标
// 2. Hover 到 user-operation → 灰色图标变为高亮
// 3. 点击高亮的图标 → 显示下拉菜单
func (m *MessageAction) clickDeleteButton(item *rod.Element) error {
	logrus.Debugf("[MessageAction.clickDeleteButton] ========== 开始点击删除按钮 ==========")

	// 调试：打印 item 的 HTML 结构
	itemHTML, _ := item.HTML()
	if len(itemHTML) > 300 {
		itemHTML = itemHTML[:300] + "..."
	}
	logrus.Debugf("[MessageAction.clickDeleteButton] item HTML: %s", itemHTML)

	// 策略1: 使用 JS 触发 mouseenter 事件 + 模拟点击
	logrus.Infof("[MessageAction.clickDeleteButton] 策略1: 使用 JS 触发 mouseenter 事件")

	// 简化的 JS：直接触发事件并点击，不使用任何数组方法
	jsClickUserOp := `(function() {
		var result = { success: false, message: '' };

		// 查找 user-operation
		var userOp = document.querySelector('[role="listitem"] .user-operation');
		if (!userOp) {
			userOp = document.querySelector('.friend-item .user-operation');
		}
		if (!userOp) {
			userOp = document.querySelector('li .user-operation');
		}
		if (!userOp) {
			userOp = document.querySelector('.user-operation');
		}

		if (!userOp) {
			return { success: false, message: '未找到 user-operation' };
		}

		// 触发 mouseenter 事件
		var event1 = new MouseEvent('mouseenter', { bubbles: true, cancelable: true, view: window });
		userOp.dispatchEvent(event1);

		var event2 = new MouseEvent('mouseover', { bubbles: true, cancelable: true, view: window });
		userOp.dispatchEvent(event2);

		// 等待 200ms 让图标出现
		var startTime = Date.now();
		while (Date.now() - startTime < 500) {
			// 查找高亮的图标
			var hoverIcon = userOp.querySelector('.list-operate-hover');
			if (!hoverIcon) {
				hoverIcon = userOp.querySelector('img.icon-operate.list-operate-hover');
			}
			if (!hoverIcon) {
				hoverIcon = userOp.querySelector('img.list-operate-hover');
			}

			if (hoverIcon) {
				hoverIcon.click();
				return { success: true, message: '点击高亮图标成功' };
			}
		}

		// 如果高亮图标找不到，点击 user-operation 内的 img
		var img = userOp.querySelector('img');
		if (img) {
			img.click();
			return { success: true, message: '点击 img 成功' };
		}

		return { success: false, message: '未找到可点击元素' };
	})()`

	jsResult, err := m.page.Eval(jsClickUserOp)
	if err != nil {
		logrus.Errorf("[MessageAction.clickDeleteButton] JS 执行失败: %v", err)
	} else {
		resultStr := jsResult.Value.String()
		logrus.Infof("[MessageAction.clickDeleteButton] JS 执行结果: %s", resultStr)

		if strings.Contains(resultStr, `"success":true`) || strings.Contains(resultStr, `"success": true`) {
			time.Sleep(800 * time.Millisecond)
			// 调用 clickDeleteFromMenu 查找删除按钮
			if err := m.clickDeleteFromMenu(); err == nil {
				return nil
			}
		}
	}

	// 策略2: 使用 go-rod 的 Hover + Click
	logrus.Warnf("[MessageAction.clickDeleteButton] 策略1失败，使用策略2: go-rod Hover + Click")

	// 先 hover 到 item
	item.Hover()
	time.Sleep(500 * time.Millisecond)

	// 获取 user-operation 元素
	var userOpEl *rod.Element
	userOpSelectors := []string{
		".user-operation",
		"[class*='user-operation']",
	}

	for _, selector := range userOpSelectors {
		els, _ := item.Elements(selector)
		if len(els) > 0 {
			userOpEl = els[0]
			logrus.Debugf("[MessageAction.clickDeleteButton] 找到 user-operation")
			break
		}
	}

	if userOpEl != nil {
		// Hover 到 user-operation
		userOpEl.Hover()
		time.Sleep(800 * time.Millisecond)

		// 尝试点击 list-operate-hover（高亮图标）
		hoverSelectors := []string{
			".list-operate-hover",
			"img.list-operate-hover",
			"img.icon-operate.list-operate-hover",
		}

		for _, sel := range hoverSelectors {
			els, _ := userOpEl.Elements(sel)
			if len(els) > 0 {
				logrus.Infof("[MessageAction.clickDeleteButton] 点击高亮图标: %s", sel)
				if err := els[0].Click(proto.InputMouseButtonLeft, 1); err == nil {
					time.Sleep(800 * time.Millisecond)
					if err := m.clickDeleteFromMenu(); err == nil {
						return nil
					}
				}
			}
		}

		// 如果高亮图标找不到，点击 user-operation 内的任何 img
		els, _ := userOpEl.Elements("img")
		if len(els) > 0 {
			logrus.Infof("[MessageAction.clickDeleteButton] 点击 user-operation 内的 img")
			if err := els[0].Click(proto.InputMouseButtonLeft, 1); err == nil {
				time.Sleep(800 * time.Millisecond)
				if err := m.clickDeleteFromMenu(); err == nil {
					return nil
				}
			}
		}
	}

	// 策略3: 直接点击 item 元素
	logrus.Warnf("[MessageAction.clickDeleteButton] 策略2失败，使用策略3: 直接点击 item")
	item.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(500 * time.Millisecond)

	return m.clickDeleteFromMenu()
}

// clickDeleteFromPage 从页面范围查找并点击删除按钮
func (m *MessageAction) clickDeleteFromPage(item *rod.Element) error {
	logrus.Debugf("[MessageAction.clickDeleteFromPage] ========== 从页面范围查找删除按钮 ==========")

	// 增加重试机制
	maxRetries := 2
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		// 查找页面上的下拉菜单或弹出框
		popupSelectors := []string{
			".operation-container",
			"[class*='dropdown-menu']",
			"[class*='dropdown']",
			"[class*='popup']",
			"[class*='menu']",
			"[class*='operate-menu']",
			"[class*='popover']",
		}

		for _, selector := range popupSelectors {
			popups, err := m.page.Elements(selector)
			if err == nil {
				logrus.Debugf("[MessageAction.clickDeleteFromPage] 选择器 %s 找到 %d 个弹出框", selector, len(popups))
				for _, popup := range popups {
					// 使用 JS 检查是否可见
					jsVisible := `(function() {
						var style = window.getComputedStyle(this);
						return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
					})`
					result, err := popup.Eval(jsVisible)
					if err != nil || !result.Value.Bool() {
						continue
					}

					// 在弹出框中找删除按钮
					btns, _ := popup.Elements("button, a, [role='button'], li, [class*='item']")
					for _, btn := range btns {
						text, _ := btn.Text()
						text = strings.TrimSpace(text)
						logrus.Debugf("[MessageAction.clickDeleteFromPage] 弹出框内按钮文本: '%s'", text)
						if strings.Contains(text, "删除") {
							logrus.Infof("[MessageAction.clickDeleteFromPage] 点击弹出菜单中的删除按钮: '%s'", text)
							return btn.Click(proto.InputMouseButtonLeft, 1)
						}
					}
				}
			}
		}
	}

	logrus.Warnf("[MessageAction.clickDeleteFromPage] 经过 %d 次重试仍未找到删除按钮", maxRetries)
	return errors.New("未找到删除按钮")
}

// clickDeleteFromMenu 点击三个点图标后出现的下拉菜单中的删除按钮
func (m *MessageAction) clickDeleteFromMenu() error {
	logrus.Debugf("[MessageAction.clickDeleteFromMenu] ========== 查找下拉菜单中的删除按钮 ==========")

	// 增加重试机制：等待菜单出现（最多4次，每次间隔800ms）
	maxRetries := 4
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			logrus.Debugf("[MessageAction.clickDeleteFromMenu] 第 %d 次重试，等待菜单出现...", retry+1)
			time.Sleep(800 * time.Millisecond)
		}

		// 调试：检查页面上所有可能的菜单/弹出元素
		jsCheck := `(function() {
			var containers = document.querySelectorAll('[class*="dropdown"], [class*="menu"], [class*="popup"], [class*="popover"], .operation-container, .operate-menu');
			var result = [];
			for (var i = 0; i < Math.min(containers.length, 10); i++) {
				var style = window.getComputedStyle(containers[i]);
				var isVisible = style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
				if (isVisible) {
					result.push({
						index: i,
						className: containers[i].className,
						display: style.display,
						visibility: style.visibility,
						opacity: style.opacity,
						html: containers[i].outerHTML.substring(0, 500)
					});
				}
			}
			return { visibleCount: result.length, items: result };
		})()`
		opResult, err := m.page.Eval(jsCheck)
		if err == nil {
			logrus.Debugf("[MessageAction.clickDeleteFromMenu] 可见菜单元素 (尝试 %d): %s", retry+1, opResult.Value.String())
		}

		// 输出当前全部的 html 内容
		_, err = m.page.HTML()
		if err == nil {
			debug.WritePageHTMLToFile(m.page, "delete_menu.html")
		}

		// 策略1: 直接在页面范围内查找可见的菜单
		foundDelete := m.findDeleteInVisibleMenus()
		if foundDelete {
			return nil
		}

		// 策略2: 尝试点击 body 空白处关闭可能存在的其他菜单，然后重新触发
		if retry == 0 {
			logrus.Debugf("[MessageAction.clickDeleteFromMenu] 策略2: 点击 body 空白处关闭其他菜单")
			bodyClick := `(function() {
				document.body.click();
				return 'body clicked';
			})()`
			m.page.Eval(bodyClick)
			time.Sleep(300 * time.Millisecond)
		}
	}

	// 所有重试都失败，打印最终页面状态
	logrus.Warnf("[MessageAction.clickDeleteFromMenu] 经过 %d 次重试仍未找到删除按钮", maxRetries)

	// 打印页面中所有可能的操作按钮
	jsAllButtons := `(function() {
		var btns = document.querySelectorAll('button, a, [role="button"], [role="menuitem"]');
		var result = [];
		for (var i = 0; i < Math.min(btns.length, 30); i++) {
			var style = window.getComputedStyle(btns[i]);
			if (style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0') {
				result.push({
					text: btns[i].innerText.substring(0, 50),
					className: btns[i].className,
					display: style.display
				});
			}
		}
		return result;
	})()`
	btnResult, err := m.page.Eval(jsAllButtons)
	if err == nil {
		logrus.Debugf("[MessageAction.clickDeleteFromMenu] 页面可见按钮列表: %s", btnResult.Value.String())
	}

	return errors.New("未找到删除菜单项")
}

// findDeleteInVisibleMenus 在页面可见的菜单中查找并点击删除按钮
func (m *MessageAction) findDeleteInVisibleMenus() bool {
	// 下拉菜单可能的选择器
	menuSelectors := []string{
		".ui-dropmenu-list",    // BOSS直聘：消息操作下拉菜单
		".operation-container", // 动态生成的下拉容器
		"[class*='dropdown-menu']",
		"[class*='operate-menu']",
		"[class*='user-menu']",
		"[class*='action-menu']",
		"[class*='context-menu']",
		"[class*='popover']",
		"[class*='popup']",
		"ul[class*='menu']",
		"ul.more-setting", // BOSS直聘：消息菜单 ul.more-setting
		"div[class*='menu']",
		"[class*='operate']",
		"[class*='action']",
	}

	for _, menuSel := range menuSelectors {
		menus, err := m.page.Elements(menuSel)
		if err == nil {
			logrus.Debugf("[MessageAction.findDeleteInVisibleMenus] 选择器 %s 找到 %d 个菜单", menuSel, len(menus))
			for _, menu := range menus {
				if menu == nil {
					continue
				}

				// 检查菜单是否可见
				jsVisible := `(function() {
					var style = window.getComputedStyle(this);
					return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
				})`
				result, err := menu.Eval(jsVisible)
				if err != nil || !result.Value.Bool() {
					continue
				}

				// 打印菜单 HTML 用于调试
				menuHTML, _ := menu.HTML()
				if len(menuHTML) > 200 {
					logrus.Debugf("[MessageAction.findDeleteInVisibleMenus] 菜单 %s HTML预览: %s...", menuSel, menuHTML[:min(200, len(menuHTML))])
				}

				// 查找菜单项（多种选择器组合）
				itemSelectors := []string{
					"ul.more-setting li",  // BOSS直聘：优先使用 more-setting
					"li[data-v-ed616276]", // BOSS直聘：带 data-v 属性的 li
					"li",
					"[class*='menu-item']",
					"[class*='dropdown-item']",
					"[class*='item']",
					"button",
					"a",
					"[role='menuitem']",
					"span",
					"div",
				}

				// 策略1: 直接查找包含"删除"文字的 span 元素（BOSS直聘结构）
				deleteSpanSelectors := []string{
					"span:text('删除')", // go-rod 内置文本选择器
					"span:has-text('删除')",
				}
				for _, sel := range deleteSpanSelectors {
					spans, err := menu.Elements(sel)
					if err == nil {
						for _, span := range spans {
							text, _ := span.Text()
							if strings.Contains(text, "删除") {
								logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 找到删除 span: %s", text)
								// 点击 span 的父级 li（删除按钮的实际可点击区域）
								if li, err := span.Element("xpath/.."); err == nil {
									if err := li.Click(proto.InputMouseButtonLeft, 1); err == nil {
										logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 点击删除 li 成功")
										return true
									}
								}
								// 直接点击 span
								if err := span.Click(proto.InputMouseButtonLeft, 1); err == nil {
									logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 点击删除 span 成功")
									return true
								}
							}
						}
					}
				}

				// 策略2: 遍历所有菜单项查找"删除"
				for _, itemSel := range itemSelectors {
					items, _ := menu.Elements(itemSel)
					for _, menuItem := range items {
						text, _ := menuItem.Text()
						text = strings.TrimSpace(text)
						if text == "" {
							continue
						}
						logrus.Debugf("[MessageAction.findDeleteInVisibleMenus] 菜单项文本: '%s'", text)

						// 查找包含"删除"文字的菜单项
						if strings.Contains(text, "删除") {
							logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 点击删除菜单项: '%s'", text)
							if err := menuItem.Click(proto.InputMouseButtonLeft, 1); err == nil {
								return true
							}
							// 如果点击失败，尝试 JS 点击
							jsClick := `(function() { this.click(); })`
							if _, err := menuItem.Eval(jsClick); err == nil {
								logrus.Infof("[MessageAction.findDeleteInVisibleMenus] JS点击删除菜单项成功")
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
