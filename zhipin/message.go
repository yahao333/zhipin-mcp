package zhipin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
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

// clickDeleteButton 在消息项中点击删除按钮
func (m *MessageAction) clickDeleteButton(item *rod.Element) error {
	logrus.Debugf("[MessageAction.clickDeleteButton] 开始查找删除按钮")

	// 尝试多种选择器定位删除按钮
	deleteSelectors := []string{
		".user-operation",           // 用户操作容器
		"[class*='user-operation']", // 包含 user-operation 的元素
		"[class*='operate']",        // 包含 operate 的元素
		".icon-operate",             // 操作图标
		"[class*='icon-operate']",   // 包含 icon-operate 的元素
	}

	var deleteBtn *rod.Element

	for _, selector := range deleteSelectors {
		logrus.Debugf("[MessageAction.clickDeleteButton] 尝试选择器: %s", selector)
		els, err := item.Elements(selector)
		if err == nil && len(els) > 0 {
			logrus.Debugf("[MessageAction.clickDeleteButton] 选择器 %s 找到 %d 个元素", selector, len(els))
			// 找到操作按钮容器
			deleteBtn = els[0]
			break
		}
	}

	if deleteBtn == nil {
		logrus.Warnf("[MessageAction.clickDeleteButton] 未找到操作按钮，尝试在父元素中查找")
		// 尝试在整个消息列表区域查找
		operateSelectors := []string{
			".friend-item:hover .user-operation",
			"[role='listitem']:hover .user-operation",
			".chat-item:hover [class*='operate']",
		}
		for _, sel := range operateSelectors {
			els, err := m.page.Elements(sel)
			if err == nil && len(els) > 0 {
				deleteBtn = els[0]
				break
			}
		}
	}

	if deleteBtn == nil {
		// 最后尝试通过文本定位删除按钮
		logrus.Debugf("[MessageAction.clickDeleteButton] 尝试通过文本查找删除按钮")
		return m.clickDeleteByText()
	}

	// 获取按钮的 HTML 用于调试
	if html, err := deleteBtn.HTML(); err == nil {
		logrus.Debugf("[MessageAction.clickDeleteButton] 操作按钮HTML长度: %d", len(html))
		if len(html) > 300 {
			html = html[:300] + "..."
		}
		logrus.Debugf("[MessageAction.clickDeleteButton] 操作按钮HTML: %s", html)
	}

	// 鼠标悬停到操作按钮以显示删除选项
	logrus.Debugf("[MessageAction.clickDeleteButton] 鼠标悬停到操作按钮")
	if err := deleteBtn.Hover(); err != nil {
		logrus.Warnf("[MessageAction.clickDeleteButton] Hover 操作按钮失败: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 查找删除按钮（可能是带有特定class或者文本的按钮）
	deleteBtnSelectors := []string{
		"img[src*='delete']",
		"[class*='delete']",
		"button[class*='delete']",
		".icon-operate-hover",        // 删除图标悬停状态
		"img[src*='operater-hover']", // 操作图标悬停状态
	}

	for _, sel := range deleteBtnSelectors {
		btns, err := deleteBtn.Elements(sel)
		if err == nil && len(btns) > 0 {
			logrus.Debugf("[MessageAction.clickDeleteButton] 找到删除按钮候选: %s", sel)
			// 第一个是置顶，第二个是删除
			if len(btns) >= 2 {
				logrus.Infof("[MessageAction.clickDeleteButton] 点击第2个按钮（删除）")
				return btns[1].Click(proto.InputMouseButtonLeft, 1)
			}
			// 如果只有一个，可能是删除按钮
			return btns[0].Click(proto.InputMouseButtonLeft, 1)
		}
	}

	// 尝试直接点击操作按钮区域
	logrus.Debugf("[MessageAction.clickDeleteButton] 尝试直接点击操作区域")
	return deleteBtn.Click(proto.InputMouseButtonLeft, 1)
}

// clickDeleteByText 通过文本查找并点击删除按钮
func (m *MessageAction) clickDeleteByText() error {
	// 尝试查找页面上的删除相关元素
	allImgs, err := m.page.Elements("img")
	if err != nil {
		return errors.New("未找到任何图片元素")
	}

	logrus.Debugf("[MessageAction.clickDeleteByText] 页面共有 %d 个图片元素", len(allImgs))

	for i, img := range allImgs {
		src, _ := img.Attribute("src")
		if src != nil && strings.Contains(*src, "operater-hover") {
			logrus.Debugf("[MessageAction.clickDeleteByText] 找到操作图标 (第 %d 个): %s", i, *src)
			// 这个应该是操作按钮
			if err := img.Click(proto.InputMouseButtonLeft, 1); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)

			// 点击后应该出现下拉菜单，查找删除选项
			menuSelectors := []string{
				"[class*='dropdown'] a",
				"[class*='menu'] li",
				"[class*='popup'] a",
			}
			for _, sel := range menuSelectors {
				menuItems, _ := m.page.Elements(sel)
				for _, item := range menuItems {
					text, _ := item.Text()
					if strings.Contains(text, "删除") {
						logrus.Infof("[MessageAction.clickDeleteByText] 点击删除菜单项")
						return item.Click(proto.InputMouseButtonLeft, 1)
					}
				}
			}
		}
	}

	return errors.New("未找到删除按钮")
}
