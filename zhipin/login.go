package zhipin

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func navigateAndWait(ctx context.Context, page *rod.Page, url string) (*rod.Page, error) {
	// navigateAndWait 用于“打开指定 URL 并等待页面稳定可用”：
	// 1) 绑定 ctx：便于上层统一控制取消/超时
	// 2) 设置页面级超时：防止卡在网络/页面事件上导致流程挂死
	// 3) 监听 NetworkRequestWillBeSent：记录重定向链路，方便排查跳转/风控/验证码页面
	// 4) 等待 DOMContentLoaded：确保页面主结构加载完成
	// 5) 等待 RequestIdle：确保页面请求进入短暂空闲，减少后续元素查找不稳定
	// 6) 拉取 PageInfo：拿到最终 URL、Title 等信息并做基础“疑似反爬/风控”判断
	const pageTimeout = 45 * time.Second
	startAt := time.Now()

	pp := page.Context(ctx).Timeout(pageTimeout)

	baseLog := logrus.WithFields(logrus.Fields{
		"url":        url,
		"timeout_ms": pageTimeout.Milliseconds(),
		"ctx_deadline": func() string {
			if d, ok := ctx.Deadline(); ok {
				return d.Format(time.RFC3339Nano)
			}
			return ""
		}(),
	})
	baseLog.Debug("navigate start")

	type redirectHop struct {
		From   string
		To     string
		Status int
	}

	var (
		mu        sync.Mutex
		redirects []redirectHop
	)

	monitorPage, cancelMonitor := pp.WithCancel()
	defer cancelMonitor()

	baseLog.Debug("redirect monitor start")
	go monitorPage.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		if e == nil || e.RedirectResponse == nil {
			return
		}
		mu.Lock()
		redirects = append(redirects, redirectHop{
			From:   e.RedirectResponse.URL,
			To:     e.Request.URL,
			Status: e.RedirectResponse.Status,
		})
		redirectCount := len(redirects)
		mu.Unlock()
		baseLog.WithFields(logrus.Fields{
			"from":   e.RedirectResponse.URL,
			"to":     e.Request.URL,
			"status": e.RedirectResponse.Status,
			"count":  redirectCount,
		}).Debug("redirect hop captured")
	})()

	waitNav := pp.WaitNavigation(proto.PageLifecycleEventNameDOMContentLoaded)
	baseLog.Debug("navigate begin")
	if err := pp.Navigate(url); err != nil {
		baseLog.WithError(err).WithField("elapsed_ms", time.Since(startAt).Milliseconds()).Debug("navigate failed")
		return nil, errors.Wrapf(err, "navigate to %s failed", url)
	}
	baseLog.Debug("wait DOMContentLoaded begin")
	waitNav()

	baseLog.WithField("elapsed_ms", time.Since(startAt).Milliseconds()).Debug("DOMContentLoaded done")

	baseLog.WithField("idle_quiet_ms", int64(500*time.Millisecond/time.Millisecond)).Debug("wait request idle begin")
	waitIdle := pp.WaitRequestIdle(500*time.Millisecond, nil, nil, nil)
	waitIdle()

	baseLog.WithField("elapsed_ms", time.Since(startAt).Milliseconds()).Debug("*page loaded*")

	info, err := pp.Info()
	if err != nil {
		baseLog.WithError(err).WithField("elapsed_ms", time.Since(startAt).Milliseconds()).Debug("get page info failed")
		return nil, errors.Wrap(err, "get page info failed")
	}

	mu.Lock()
	redirectCount := len(redirects)
	redirectCopy := make([]redirectHop, redirectCount)
	copy(redirectCopy, redirects)
	mu.Unlock()

	finalURL := info.URL
	isRedirected := !strings.HasPrefix(finalURL, url)

	baseLog.WithFields(logrus.Fields{
		"final_url":  finalURL,
		"title":      info.Title,
		"redirected": isRedirected,
	}).Debug("page info collected")

	isAntiBot := false
	reason := ""
	suspiciousKeywords := []string{
		"captcha",
		"verify",
		"geetest",
		"challenge",
		"security",
		"risk",
		"validate",
		"robot",
	}
	lowerFinalURL := strings.ToLower(finalURL)
	lowerTitle := strings.ToLower(info.Title)
	for _, kw := range suspiciousKeywords {
		if strings.Contains(lowerFinalURL, kw) || strings.Contains(lowerTitle, kw) {
			isAntiBot = true
			reason = kw
			break
		}
	}
	if !isAntiBot {
		if strings.Contains(info.Title, "验证") || strings.Contains(info.Title, "安全") || strings.Contains(info.Title, "机器人") {
			isAntiBot = true
			reason = "title"
		}
	}

	logrus.WithFields(logrus.Fields{
		"url":            url,
		"final_url":      finalURL,
		"title":          info.Title,
		"redirected":     isRedirected,
		"redirect_count": redirectCount,
		"elapsed_ms":     time.Since(startAt).Milliseconds(),
		"anti_bot":       isAntiBot,
		"anti_bot_hint":  reason,
		"redirects":      redirectCopy,
	}).Debug("navigate done")

	return pp, nil
}

// Login 登录操作
type Login struct {
	page *rod.Page
}

// NewLogin 创建登录操作
func NewLogin(page *rod.Page) *Login {
	return &Login{page: page}
}

// CheckLoginStatus 检查登录状态
func (l *Login) CheckLoginStatus(ctx context.Context) (bool, error) {
	logrus.Debugf("check login status")
	// 访问BOSS直聘首页
	pp, err := navigateAndWait(ctx, l.page, "https://www.zhipin.com/")
	if err != nil {
		return false, err
	}
	logrus.Debugf("page loaded")

	// 等待页面稳定
	time.Sleep(time.Second)

	// 检查是否有登录按钮（未登录）
	exists, _, err := pp.Has(".btns .header-login-btn")
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	logrus.Debugf("login button exists: %v", exists)
	if exists {
		return false, nil
	}

	// 检查是否已登录（通过检查用户头像或用户名元素）
	// 登录成功后有 <div class="user-nav"> 下的 <li class="nav-figure">
	exists, _, err = pp.Has(".user-name, .nick-name, .boss-avatar, .nav-figure, .user-nav")
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	if exists {
		return true, nil
	}

	logrus.Debugf("user is not logged in")

	// 有登录按钮说明未登录
	return !exists, nil
}

// FetchQrcodeImage 获取登录二维码
func (l *Login) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	logrus.Debugf("fetch qrcode image")
	// 访问BOSS直聘登录页
	pp, err := navigateAndWait(ctx, l.page, "https://www.zhipin.com/user/login.html")
	if err != nil {
		return "", false, err
	}
	logrus.Debugf("login page loaded")

	// 等待二维码加载
	time.Sleep(5 * time.Second)

	// 检查是否已经登录
	exists, _, err := pp.Has(".user-name, .nick-name, .boss-avatar")
	logrus.Debugf("user is logged in: %v %v", exists, err)
	if err != nil {
		return "", false, errors.Wrap(err, "check login status failed")
	}
	if exists {
		return "", true, nil
	}

	// 获取二维码图片 - 尝试多个选择器（优先选择 img 标签）
	selectors := []string{
		".qr-code-box .qr-img-box img",
		".qr-img-box img",
		".qrcode img",
		".login-qrcode img",
		"#qrcode img",
		"[class*='qrcode'] img",
	}
	var el *rod.Element

	for _, sel := range selectors {
		el, err = pp.Timeout(5 * time.Second).Element(sel)
		if err == nil {
			break
		}
	}

	if el == nil || err != nil {
		return "", false, errors.Wrap(err, "get qrcode failed")
	}

	src, err := el.Attribute("src")
	if err != nil {
		return "", false, errors.Wrap(err, "get qrcode src failed")
	}
	if src == nil || len(*src) == 0 {
		return "", false, errors.New("qrcode src is empty")
	}

	return *src, false, nil
}

// FetchQrcodeImageAsBase64 获取登录二维码图片（返回 base64）
func (l *Login) FetchQrcodeImageAsBase64(ctx context.Context) (string, bool, error) {
	// 1. 获取相对路径（复用现有逻辑）
	src, loggedIn, err := l.fetchQrcodeSrc(ctx)
	if err != nil || loggedIn {
		return "", loggedIn, err
	}

	// 2. 拼接完整 URL
	fullURL := "https://www.zhipin.com" + src

	// 3. 下载图片
	imgData, err := l.downloadImage(ctx, fullURL)
	if err != nil {
		return "", false, errors.Wrap(err, "download qrcode image failed")
	}

	// 4. 转换为 base64
	base64Str := base64.StdEncoding.EncodeToString(imgData)
	base64WithPrefix := "data:image/png;base64," + base64Str

	return base64WithPrefix, false, nil
}

// fetchQrcodeSrc 内部方法：获取二维码相对路径
func (l *Login) fetchQrcodeSrc(ctx context.Context) (string, bool, error) {
	// 访问BOSS直聘登录页
	pp, err := navigateAndWait(ctx, l.page, "https://www.zhipin.com/user/login.html")
	if err != nil {
		return "", false, err
	}

	// 等待二维码加载
	time.Sleep(5 * time.Second)

	// 检查是否已经登录
	exists, _, err := pp.Has(".user-name, .nick-name, .boss-avatar")
	if err != nil {
		return "", false, err
	}
	if exists {
		return "", true, nil
	}

	// 获取二维码相对路径
	selectors := []string{
		".qr-code-box .qr-img-box img",
		".qr-img-box img",
		".qrcode img",
		".login-qrcode img",
		"#qrcode img",
		"[class*='qrcode'] img",
	}
	var el *rod.Element

	for _, sel := range selectors {
		el, err = pp.Timeout(5 * time.Second).Element(sel)
		if err == nil {
			break
		}
	}

	if el == nil || err != nil {
		return "", false, errors.Wrap(err, "get qrcode failed")
	}

	src, err := el.Attribute("src")
	if err != nil || src == nil || len(*src) == 0 {
		return "", false, errors.New("qrcode src is empty")
	}

	return *src, false, nil
}

// downloadImage 下载图片
func (l *Login) downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// 添加必要的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// WaitForLogin 等待扫码登录成功
func (l *Login) WaitForLogin(ctx context.Context) bool {
	pp := l.page.Context(ctx)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			// 检查是否出现用户头像或用户名元素，表示登录成功
			// 从 HTML 分析：登录成功后有 <div class="user-nav"> 下的 <li class="nav-figure">
			exists, _, err := pp.Has(".user-name, .nick-name, .boss-avatar, .nav-figure, .user-nav")
			logrus.Debugf("scan login -> %v", exists)
			if err == nil && exists {
				// 检查用户名元素是否有有效文本
				if el := pp.MustElement(".nav-figure a span.label-text"); el != nil {
					if text, err := el.Text(); err == nil && len(text) > 0 {
						logrus.Debugf("登录成功检测到用户名: %s", text)
						return true
					}
				}
				// 如果没有找到用户名元素但有用户元素存在，也认为登录成功
				return true
			}
		}
	}
}

// LoginWithPassword 使用密码登录
func (l *Login) LoginWithPassword(ctx context.Context, username, password string) (*LoginResult, error) {
	// 访问BOSS直聘登录页
	pp, err := navigateAndWait(ctx, l.page, "https://www.zhipin.com/user/login.html")
	if err != nil {
		return nil, err
	}
	time.Sleep(2 * time.Second)

	// 切换到密码登录
	pp.MustElement("a[tab='account']").MustClick()
	time.Sleep(1 * time.Second)

	// 输入用户名
	pp.MustElement("#account").MustInput(username)

	// 输入密码
	pp.MustElement("#password").MustInput(password)

	// 点击登录按钮
	pp.MustElement(".btn-login").MustClick()

	// 等待登录结果
	time.Sleep(3 * time.Second)

	// 检查登录状态
	isLoggedIn, err := l.CheckLoginStatus(ctx)
	if err != nil {
		return nil, err
	}

	if isLoggedIn {
		return &LoginResult{
			Success:  true,
			Username: username,
			Message:  "登录成功",
		}, nil
	}

	// 检查是否有验证码
	html, _ := pp.HTML()
	if strings.Contains(html, "验证码") || strings.Contains(html, "verify") {
		return &LoginResult{
			Success: false,
			Message: "需要输入验证码，请使用二维码登录",
		}, nil
	}

	return &LoginResult{
		Success: false,
		Message: "登录失败，请检查账号密码或使用二维码登录",
	}, nil
}

// EnsureLoggedIn 确保已登录
func EnsureLoggedIn(page *rod.Page) error {
	login := NewLogin(page)
	ctx := context.Background()

	isLoggedIn, err := login.CheckLoginStatus(ctx)
	if err != nil {
		return err
	}

	if !isLoggedIn {
		return errors.New("请先登录")
	}

	return nil
}
