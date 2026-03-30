package zhipin

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
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
			".name",
			".person-name",
			".chat-name",
			"[class*='name']",
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
			".digest",
			".message-digest",
			".last-message",
			"[class*='digest']",
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
