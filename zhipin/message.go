package zhipin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/pkg/debug"
	"github.com/yahao333/zhipin-mcp/pkg/delay"
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

// FindMessageItem 查找匹配的消息项
// 通过 personName, companyName, jobTitle 模糊匹配消息项，返回匹配的元素
// 如果未找到返回 nil
func (m *MessageAction) FindMessageItem(personName, companyName, jobTitle string) *rod.Element {
	logrus.Debugf("[MessageAction.FindMessageItem] ========== 查找消息项 ==========")
	logrus.Debugf("[MessageAction.FindMessageItem] 筛选条件: personName=%s, companyName=%s, jobTitle=%s", personName, companyName, jobTitle)

	// 消息列表选择器
	listSelectors := []string{".friend-item", "[role='listitem']", ".chat-item", ".dialog-item", ".message-item"}

	// 人名选择器
	nameSelectors := []string{".title-box .name-text", ".name-text", ".name"}

	// 公司名选择器
	companySelectors := []string{".title-box .name-box > span:nth-child(2)", ".company-name", "[class*='company']"}

	// 职位名选择器
	jobSelectors := []string{".title-box .name-box > span:nth-child(4)", ".job-title", "[class*='job']"}

	// 遍历每个列表选择器
	for _, listSel := range listSelectors {
		items, err := m.page.Elements(listSel)
		if err != nil || len(items) == 0 {
			continue
		}
		logrus.Debugf("[MessageAction.FindMessageItem] 使用选择器 %s 找到 %d 个元素", listSel, len(items))

		// 遍历每个消息项
		for i, item := range items {
			// 查找人名称
			var nameEl *rod.Element
			for _, sel := range nameSelectors {
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
			logrus.Debugf("[MessageAction.FindMessageItem] 检查第 %d 个元素, 人名: %s", i+1, nameText)

			// 模糊匹配人名
			if personName != "" && !strings.Contains(nameText, personName) {
				continue
			}

			// 匹配公司名称
			companyMatch := true
			if companyName != "" {
				var cEl *rod.Element
				for _, sel := range companySelectors {
					if el, err := item.Element(sel); err == nil {
						cEl = el
						break
					}
				}
				if cEl != nil {
					cText, _ := cEl.Text()
					cText = strings.TrimSpace(cText)
					companyMatch = strings.Contains(cText, companyName)
					logrus.Debugf("[MessageAction.FindMessageItem] 匹配公司: %s vs %s, 匹配=%v", cText, companyName, companyMatch)
				}
			}

			// 匹配职位名称
			jobMatch := true
			if jobTitle != "" {
				var jEl *rod.Element
				for _, sel := range jobSelectors {
					if el, err := item.Element(sel); err == nil {
						jEl = el
						break
					}
				}
				if jEl != nil {
					jText, _ := jEl.Text()
					jText = strings.TrimSpace(jText)
					jobMatch = strings.Contains(jText, jobTitle)
					logrus.Debugf("[MessageAction.FindMessageItem] 匹配职位: %s vs %s, 匹配=%v", jText, jobTitle, jobMatch)
				}
			}

			// 所有条件都匹配
			if companyMatch && jobMatch {
				logrus.Infof("[MessageAction.FindMessageItem] 找到匹配的消息项，人名: %s", nameText)
				return item
			}
		}
	}

	logrus.Warnf("[MessageAction.FindMessageItem] 未找到匹配的消息项")
	return nil
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

	// 步骤1: 刷新消息列表页面
	url := "https://www.zhipin.com/web/geek/chat"
	if err := m.page.Navigate(url); err != nil {
		logrus.Errorf("[MessageAction.DeleteMessage] Navigate 失败: %v", err)
		return err
	}
	m.page.WaitLoad()
	delay.Short()

	// 步骤2: 查找匹配的消息项
	item := m.FindMessageItem(personName, companyName, jobTitle)
	if item == nil {
		logrus.Warnf("[MessageAction.DeleteMessage] 未找到匹配的消息")
		return errors.New("未找到匹配的消息")
	}

	// 步骤3: 鼠标悬停到目标元素，显示操作按钮
	if html, err := item.HTML(); err == nil {
		logrus.Debugf("[MessageAction.DeleteMessage] targetItem HTML: %s", html)
	} else {
		logrus.Debugf("[MessageAction.DeleteMessage] 获取 targetItem HTML 失败: %v", err)
	}
	logrus.Debugf("[MessageAction.DeleteMessage] 鼠标悬停到消息项")

	err := item.Hover()
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
// 1. Hover 到消息项 → 显示灰色的三个点图标 (.list-operate)
// 2. Hover 到灰色图标 → CSS 切换到高亮图标 (.list-operate-hover)
// 3. 点击高亮图标 → 弹出菜单
func (m *MessageAction) clickDeleteButton(item *rod.Element) error {
	logrus.Debugf("[MessageAction.clickDeleteButton] ========== 开始点击删除按钮 ==========")

	// 策略1: 使用 JS 直接操作（最可靠）
	// JS 可以直接触发正确的鼠标事件序列
	logrus.Infof("[MessageAction.clickDeleteButton] 策略1: 使用 JS 直接点击")

	jsClick := `(function() {
		try {
			// 查找目标 item 内的 user-operation
			var userOp = this.querySelector('.user-operation');
			if (!userOp) {
				return '未找到 user-operation';
			}

			// 触发正确的鼠标事件序列
			var mouseEvents = ['mouseenter', 'mouseover', 'mousein'];
			for (var i = 0; i < mouseEvents.length; i++) {
				var evt = new MouseEvent(mouseEvents[i], {
					bubbles: true,
					cancelable: true,
					view: window
				});
				userOp.dispatchEvent(evt);
			}

			// 查找并点击图标
			var icon = userOp.querySelector('.list-operate');
			if (!icon) {
				icon = userOp.querySelector('.list-operate-hover');
			}
			if (!icon) {
				icon = userOp.querySelector('img');
			}

			if (icon) {
				icon.click();
				return '点击图标成功';
			}

			// 回退：点击 user-operation 本身
			userOp.click();
			return '点击 user-operation 成功';
		} catch (e) {
			return 'JS错误: ' + e.message;
		}
	})`

	jsResult, err := item.Eval(jsClick)
	if err != nil {
		logrus.Errorf("[MessageAction.clickDeleteButton] JS 执行失败: %v", err)
	} else if jsResult != nil {
		resultStr := jsResult.Value.String()
		logrus.Infof("[MessageAction.clickDeleteButton] JS 结果: %s", resultStr)
	}

	// 等待菜单出现
	delay.Short()

	// 处理菜单
	if m.clickDeleteFromMenu() == nil {
		return nil
	}

	// 策略2: 使用 go-rod hover + 点击灰色图标
	logrus.Warnf("[MessageAction.clickDeleteButton] 策略2: 使用 go-rod hover + 点击")

	item.Hover()
	delay.Short()

	userOpEl, err := item.Element(".user-operation")
	if err != nil {
		logrus.Errorf("[MessageAction.clickDeleteButton] 未找到 user-operation: %v", err)
		return err
	}

	userOpEl.Hover()
	delay.Short()

	// 尝试点击灰色图标 (.list-operate)
	grayIcon, err := userOpEl.Element(".list-operate")
	if err != nil {
		logrus.Debugf("[MessageAction.clickDeleteButton] 未找到灰色图标，尝试 img")
		grayIcon, err = userOpEl.Element("img")
	}

	if grayIcon != nil {
		logrus.Infof("[MessageAction.clickDeleteButton] 点击图标")
		if err := grayIcon.Click(proto.InputMouseButtonLeft, 1); err != nil {
			logrus.Errorf("[MessageAction.clickDeleteButton] 点击失败: %v", err)
		} else {
			delay.Short()
			if m.clickDeleteFromMenu() == nil {
				return nil
			}
		}
	}

	// 策略3: 直接点击 user-operation
	logrus.Warnf("[MessageAction.clickDeleteButton] 策略3: 直接点击 user-operation")
	if err := userOpEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}
	delay.Short()

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

				// 策略1: 使用 JS 直接触发删除点击，然后处理确认弹窗
				// BOSS直聘删除后会弹出确认对话框
				logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 策略1: 使用 JS 触发删除点击")
				jsClickDelete := `(function() {
					var deleteItem = null;
					var spans = document.querySelectorAll('ul.more-setting span');
					for (var i = 0; i < spans.length; i++) {
						if (spans[i].textContent.trim() === '删除') {
							// 找到 span 后，触发其父级 li 的点击
							var li = spans[i].parentElement;
							if (li && li.tagName === 'LI') {
								li.click();
								deleteItem = 'li_clicked';
							} else {
								// 直接点击 span
								spans[i].click();
								deleteItem = 'span_clicked';
							}
							break;
						}
					}
					return deleteItem || 'not_found';
				})()`
				clickResult, err := menu.Eval(jsClickDelete)
				if err == nil {
					logrus.Infof("[MessageAction.findDeleteInVisibleMenus] JS点击结果: %s", clickResult.Value.String())
					if clickResult.Value.String() != "not_found" {
						// 等待确认弹窗出现
						time.Sleep(800 * time.Millisecond)
						// 处理确认弹窗
						if m.handleDeleteConfirm() {
							return true
						}
					}
				}

				// 策略2: 遍历所有菜单项查找"删除"，使用 JS 点击
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
							logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 使用 JS 点击删除菜单项: '%s'", text)
							// 使用 JS 点击（更可靠）
							jsClick := `(function() {
								// 如果是 li，直接点击
								if (this.tagName === 'LI') {
									this.click();
									return 'li_clicked';
								}
								// 否则点击父级 li
								if (this.parentElement && this.parentElement.tagName === 'LI') {
									this.parentElement.click();
									return 'parent_li_clicked';
								}
								// 回退：直接点击
								this.click();
								return 'clicked';
							})`
							if _, err := menuItem.Eval(jsClick); err == nil {
								logrus.Infof("[MessageAction.findDeleteInVisibleMenus] JS点击删除成功")
								delay.Short()
								if m.handleDeleteConfirm() {
									return true
								}
							} else {
								// 回退：使用鼠标点击
								if err := menuItem.Click(proto.InputMouseButtonLeft, 1); err == nil {
									logrus.Infof("[MessageAction.findDeleteInVisibleMenus] 鼠标点击删除成功")
									delay.Short()
									if m.handleDeleteConfirm() {
										return true
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

// handleDeleteConfirm 处理删除确认对话框
// BOSS直聘删除消息后会弹出确认对话框，需要点击确认
func (m *MessageAction) handleDeleteConfirm() bool {
	logrus.Debugf("[MessageAction.handleDeleteConfirm] 检查是否有删除确认对话框...")

	// 等待对话框出现
	delay.Short()

	// 先输出页面上的弹窗信息用于调试
	jsCheckPopup := `(function() {
		var result = {
			modals: [],
			buttons: []
		};
		// 查找所有可能包含确认按钮的弹窗
		var popups = document.querySelectorAll('[class*="modal"], [class*="dialog"], [class*="confirm"], [class*="tip"], [class*="popup"], [role="dialog"]');
		for (var i = 0; i < popups.length; i++) {
			var style = window.getComputedStyle(popups[i]);
			var isVisible = style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
			if (isVisible) {
				result.modals.push({
					className: popups[i].className,
					html: popups[i].outerHTML.substring(0, 400)
				});
				// 查找弹窗中的按钮
				var btns = popups[i].querySelectorAll('button, [class*="btn"], a');
				for (var j = 0; j < btns.length; j++) {
					result.buttons.push({
						text: btns[j].textContent.trim(),
						className: btns[j].className
					});
				}
			}
		}
		return result;
	})()`
	popupResult, err := m.page.Eval(jsCheckPopup)
	if err == nil && popupResult != nil {
		logrus.Debugf("[MessageAction.handleDeleteConfirm] 弹窗信息: %s", popupResult.Value.String())
	} else {
		logrus.Debugf("[MessageAction.handleDeleteConfirm] 获取弹窗信息失败: %v", err)
	}

	// 尝试使用 JS 直接处理确认弹窗（最可靠的方式）
	jsHandleConfirm := `(function() {
		// BOSS直聘对话框结构：
		// div[data-type="boss-dialog"]
		//   └── div.boss-popup__content
		//         └── div.boss-dialog__footer
		//               └── span.boss-dialog__button (取消)
		//               └── span.boss-dialog__button (确定)

		// 优先：查找 BOSS直聘确认对话框中的"确定"按钮
		var dialogContents = document.querySelectorAll('div[data-type="boss-dialog"] .boss-popup__content');
		for (var i = 0; i < dialogContents.length; i++) {
			var style = window.getComputedStyle(dialogContents[i]);
			var isVisible = style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
			if (!isVisible) continue;

			// 在 boss-dialog__footer 中查找按钮
			var footerBtns = dialogContents[i].querySelectorAll('.boss-dialog__footer span');
			for (var j = 0; j < footerBtns.length; j++) {
				var text = footerBtns[j].textContent.trim();
				// 找"确定"按钮（跳过"取消"）
				if (text === '确定' || text === '确认') {
					footerBtns[j].click();
					return 'clicked: ' + text;
				}
			}
		}

		// 回退：查找其他确认弹窗
		var popups = document.querySelectorAll('[class*="modal"], [class*="dialog"], [class*="confirm"], [role="dialog"]');
		for (var i = 0; i < popups.length; i++) {
			var style = window.getComputedStyle(popups[i]);
			var isVisible = style.display !== 'none' && style.visibility !== 'hidden';
			if (isVisible) {
				// 查找确认/确定/删除按钮
				var btns = popups[i].querySelectorAll('button, [class*="btn"], span[class*="button"]');
				for (var j = 0; j < btns.length; j++) {
					var text = btns[j].textContent.trim();
					if (text === '确定' || text === '确认' || text === 'Yes') {
						btns[j].click();
						return 'clicked: ' + text;
					}
				}
			}
		}
		return 'no_confirm_found';
	})()`
	confirmResult, err := m.page.Eval(jsHandleConfirm)
	if err != nil {
		logrus.Warnf("[MessageAction.handleDeleteConfirm] JS执行失败: %v", err)
	} else if confirmResult == nil {
		logrus.Warnf("[MessageAction.handleDeleteConfirm] JS返回结果为 nil")
	} else {
		resultStr := confirmResult.Value.String()
		logrus.Infof("[MessageAction.handleDeleteConfirm] JS确认结果: %s", resultStr)

		if resultStr != "no_confirm_found" {
			logrus.Infof("[MessageAction.handleDeleteConfirm] 确认操作完成")
			delay.Short()
			return true
		}
	}

	// 回退：使用 go-rod 查找确认按钮
	confirmSelectors := []string{
		".boss-dialog__button",      // BOSS直聘对话框按钮
		".dialog__button",           // 对话框按钮
		"[class*='dialog__button']", // 其他对话框按钮
		"span:text('确定')",           // 包含"确定"的 span
		"span:text('确认')",           // 包含"确认"的 span
		"button:text('确定')",
		"button:text('确认')",
		"[class*='btn-confirm']",
		"[class*='btn-ok']",
		".btn-primary",
	}

	for _, sel := range confirmSelectors {
		btns, err := m.page.Elements(sel)
		if err == nil && len(btns) > 0 {
			for _, btn := range btns {
				text, _ := btn.Text()
				text = strings.TrimSpace(text)
				logrus.Debugf("[MessageAction.handleDeleteConfirm] 检查按钮: %s", text)
				// 跳过"取消"按钮
				if text == "取消" {
					continue
				}
				// 点击"确定"或"确认"按钮
				if text == "确定" || text == "确认" {
					logrus.Infof("[MessageAction.handleDeleteConfirm] 点击确认按钮: %s", text)
					// 使用 JS 点击更可靠
					jsClick := `(function() { this.click(); })`
					if _, err := btn.Eval(jsClick); err == nil {
						logrus.Infof("[MessageAction.handleDeleteConfirm] JS点击确认成功")
						delay.Short()
						return true
					}
					// 回退到鼠标点击
					if err := btn.Click(proto.InputMouseButtonLeft, 1); err == nil {
						logrus.Infof("[MessageAction.handleDeleteConfirm] 点击确认成功")
						delay.Short()
						return true
					}
				}
			}
		}
	}

	logrus.Warnf("[MessageAction.handleDeleteConfirm] 未找到确认对话框")
	return false
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SendResult 发送消息结果
type SendResult struct {
	Success    bool
	PersonName string // 发送对象的姓名
	Message    string // 详细信息
}

// SendMessage 发送消息
// 通过 personName, companyName, jobTitle 定位消息项，点击打开对话框，输入内容并发送
func (m *MessageAction) SendMessage(ctx context.Context, personName, companyName, jobTitle, content string) (*SendResult, error) {
	logrus.Debugf("[MessageAction.SendMessage] ========== 开始发送消息 ==========")
	logrus.Debugf("[MessageAction.SendMessage] 筛选条件: personName=%s, companyName=%s, jobTitle=%s", personName, companyName, jobTitle)

	// 步骤1: 确保在消息列表页面
	url := "https://www.zhipin.com/web/geek/chat"
	if err := m.page.Navigate(url); err != nil {
		logrus.Errorf("[MessageAction.SendMessage] Navigate 失败: %v", err)
		return nil, err
	}
	m.page.WaitLoad()
	delay.Short()

	// 步骤2: 查找匹配的消息项（使用共用方法）
	targetItem := m.FindMessageItem(personName, companyName, jobTitle)
	if targetItem == nil {
		logrus.Errorf("[MessageAction.SendMessage] 未找到匹配的消息项: %s", personName)
		return nil, fmt.Errorf("未找到匹配的消息项: %s", personName)
	}

	// 步骤3: 点击消息项打开对话框
	logrus.Debugf("[MessageAction.SendMessage] 点击消息项打开对话框...")

	// 先确保元素可见 - 使用 JS 滚动到视图
	jsScroll := `(function() {
		this.scrollIntoView({ behavior: 'instant', block: 'center' });
		return 'scrolled';
	})`
	if _, err := targetItem.Eval(jsScroll); err != nil {
		logrus.Warnf("[MessageAction.SendMessage] JS scroll 失败: %v", err)
	}

	delay.Short()

	// 点击消息项 - 尝试 go-rod Click
	if err := targetItem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		logrus.Warnf("[MessageAction.SendMessage] Click 失败: %v", err)
		// 备选: 使用 JS click
		jsClick := `(function() {
			this.click();
			return 'clicked';
		})`
		if _, err := targetItem.Eval(jsClick); err != nil {
			logrus.Errorf("[MessageAction.SendMessage] JS click 失败: %v", err)
			return nil, fmt.Errorf("点击消息项失败: %v", err)
		}
		logrus.Infof("[MessageAction.SendMessage] 使用 JS click 成功")
	} else {
		logrus.Infof("[MessageAction.SendMessage] 使用 go-rod Click 成功")
	}

	// 等待对话框出现
	delay.Long()
	time.Sleep(3 * time.Second)

	// 调试: 保存点击后的页面 HTML
	debug.WritePageHTMLToFile(m.page, "send_message_after_click.html")

	// 步骤4: 定位输入框
	logrus.Debugf("[MessageAction.SendMessage] 定位输入框...")
	inputEl, err := m.findInputElement()
	if err != nil {
		logrus.Errorf("[MessageAction.SendMessage] 定位输入框失败: %v", err)
		return nil, fmt.Errorf("无法定位消息输入框: %v", err)
	}

	// 步骤5: 输入消息内容
	logrus.Debugf("[MessageAction.SendMessage] 输入消息内容...")
	if err := m.typeInInput(inputEl, content); err != nil {
		logrus.Errorf("[MessageAction.SendMessage] 输入内容失败: %v", err)
		return nil, fmt.Errorf("输入内容失败: %v", err)
	}

	// 步骤6: 点击发送按钮
	logrus.Debugf("[MessageAction.SendMessage] 点击发送按钮...")
	if err := m.clickSendButton(); err != nil {
		logrus.Errorf("[MessageAction.SendMessage] 点击发送按钮失败: %v", err)
		return nil, fmt.Errorf("无法定位发送按钮: %v", err)
	}

	// 等待发送完成
	delay.Short()

	// 验证发送结果
	success := m.verifySendSuccess(inputEl)
	if success {
		logrus.Infof("[MessageAction.SendMessage] ========== 发送消息成功 ==========")
		return &SendResult{
			Success:    true,
			PersonName: personName,
			Message:    "消息发送成功",
		}, nil
	}

	logrus.Warnf("[MessageAction.SendMessage] 发送结果未知，返回成功")
	return &SendResult{
		Success:    true,
		PersonName: personName,
		Message:    "消息可能已发送",
	}, nil
}

// findInputElement 定位消息输入框
func (m *MessageAction) findInputElement() (*rod.Element, error) {
	logrus.Debugf("[MessageAction.findInputElement] ========== 查找输入框 ==========")

	// 尝试多种选择器
	inputSelectors := []string{
		"textarea.msg-textarea",          // BOSS直聘标准选择器
		"textarea[class*='msg']",         // 包含 msg 的 textarea
		"textarea[class*='message']",     // 包含 message 的 textarea
		"textarea",                       // 通用 textarea
		"[class*='msg-input'] textarea",  // 输入框容器内的 textarea
		"[class*='chat-input'] textarea", // 聊天输入框容器
		"[class*='input-area'] textarea", // 输入区域
	}

	for _, selector := range inputSelectors {
		logrus.Debugf("[MessageAction.findInputElement] 尝试选择器: %s", selector)
		el, err := m.page.Element(selector)
		if err == nil {
			logrus.Debugf("[MessageAction.findInputElement] 找到输入框: %s", selector)
			return el, nil
		}
	}

	// 尝试 contenteditable div
	contentEditableSelectors := []string{
		"div[contenteditable='true']",
		"[class*='msg-input'][contenteditable='true']",
		"[class*='chat-input'][contenteditable='true']",
	}

	for _, selector := range contentEditableSelectors {
		logrus.Debugf("[MessageAction.findInputElement] 尝试 contenteditable: %s", selector)
		el, err := m.page.Element(selector)
		if err == nil {
			logrus.Debugf("[MessageAction.findInputElement] 找到 contenteditable 输入框: %s", selector)
			return el, nil
		}
	}

	return nil, fmt.Errorf("未找到输入框")
}

// typeInInput 在输入框中输入内容
func (m *MessageAction) typeInInput(inputEl *rod.Element, content string) error {
	logrus.Debugf("[MessageAction.typeInInput] 输入内容: %s", content)

	// 先尝试点击输入框获得焦点
	if err := inputEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		logrus.Warnf("[MessageAction.typeInInput] 点击输入框失败: %v", err)
	}

	delay.Short()

	// 使用 JS 直接设置输入框的值
	jsSetValue := fmt.Sprintf(`(function() {
		this.value = %q;
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
		return 'set';
	})`, content)
	if _, err := inputEl.Eval(jsSetValue); err != nil {
		return fmt.Errorf("输入内容失败: %v", err)
	}

	return nil
}

// clickSendButton 点击发送按钮
func (m *MessageAction) clickSendButton() error {
	logrus.Debugf("[MessageAction.clickSendButton] ========== 查找发送按钮 ==========")

	// 尝试多种选择器
	sendButtonSelectors := []string{
		"button.btn-send",                // BOSS直聘标准发送按钮
		"button[class*='send']",          // 包含 send 的 button
		"button[class*='msg-send']",      // 消息发送按钮
		"[class*='send-btn']",            // 发送按钮容器
		"[class*='operate-btn']",         // 操作按钮
		".chat-window button:last-child", // 聊天窗口最后一个按钮
		".btn.btn-primary",               // 主要按钮
	}

	var sendBtn *rod.Element
	var err error

	for _, selector := range sendButtonSelectors {
		logrus.Debugf("[MessageAction.clickSendButton] 尝试选择器: %s", selector)
		sendBtn, err = m.page.Element(selector)
		if err == nil {
			// 检查按钮文本是否包含"发送"
			btnText, _ := sendBtn.Text()
			if strings.Contains(btnText, "发送") || strings.Contains(btnText, "send") {
				logrus.Debugf("[MessageAction.clickSendButton] 找到发送按钮: %s (文本: %s)", selector, btnText)
				break
			}
		}
	}

	if sendBtn == nil {
		return fmt.Errorf("未找到发送按钮")
	}

	// 点击发送按钮
	if err := sendBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		logrus.Errorf("[MessageAction.clickSendButton] 点击发送按钮失败: %v", err)
		// 尝试使用 JS 点击
		jsClick := `(function() { this.click(); })`
		if _, err := sendBtn.Eval(jsClick); err != nil {
			return fmt.Errorf("点击发送按钮失败: %v", err)
		}
	}

	logrus.Debugf("[MessageAction.clickSendButton] 发送按钮已点击")
	return nil
}

// verifySendSuccess 验证发送是否成功
func (m *MessageAction) verifySendSuccess(inputEl *rod.Element) bool {
	logrus.Debugf("[MessageAction.verifySendSuccess] ========== 验证发送结果 ==========")

	// 方法1: 检查输入框是否被清空（消息已发出）
	inputValue, _ := inputEl.Attribute("value")
	if inputValue != nil && *inputValue == "" {
		logrus.Debugf("[MessageAction.verifySendSuccess] 输入框已清空，发送成功")
		return true
	}

	// 方法2: 检查是否有 toast 提示成功
	toastSelectors := []string{
		".toast",
		".message-toast",
		"[class*='toast']",
	}

	for _, selector := range toastSelectors {
		toast, err := m.page.Element(selector)
		if err == nil {
			toastText, _ := toast.Text()
			if strings.Contains(toastText, "发送成功") || strings.Contains(toastText, "成功") {
				logrus.Debugf("[MessageAction.verifySendSuccess] 检测到成功提示: %s", toastText)
				return true
			}
		}
	}

	// 方法3: 检查消息列表中该对话是否有新消息
	// 这个方法需要在发送前记录消息摘要，发送后再对比

	return false
}
