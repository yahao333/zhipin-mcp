package zhipin

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
)

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
	pp := l.page.Context(ctx)

	// 访问BOSS直聘首页
	pp.MustNavigate("https://www.zhipin.com/").MustWaitLoad()

	// 等待页面稳定
	time.Sleep(1 * time.Second)

	// 检查是否已登录（通过检查用户头像或用户名元素）
	exists, _, err := pp.Has(".user-name, .nick-name, .boss-avatar")
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	if exists {
		return true, nil
	}

	// 检查是否有登录按钮（未登录）
	exists, _, err = pp.Has(".btn-login, .login-btn")
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	// 有登录按钮说明未登录
	return !exists, nil
}

// FetchQrcodeImage 获取登录二维码
func (l *Login) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	pp := l.page.Context(ctx)

	// 访问BOSS直聘登录页
	pp.MustNavigate("https://www.zhipin.com/user/login.html").MustWaitLoad()

	// 等待二维码加载
	time.Sleep(5 * time.Second)

	// 检查是否已经登录
	exists, _, err := pp.Has(".user-name, .nick-name, .boss-avatar")
	if err != nil {
		return "", false, errors.Wrap(err, "check login status failed")
	}
	if exists {
		return "", true, nil
	}

	// 获取二维码图片 - 尝试多个选择器
	selectors := []string{".qrcode img", ".login-qrcode img", "#qrcode", ".qrcode", "[class*='qrcode'] img"}
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
			exists, _, err := pp.Has(".user-name, .nick-name, .boss-avatar")
			if err == nil && exists {
				return true
			}
		}
	}
}

// LoginWithPassword 使用密码登录
func (l *Login) LoginWithPassword(ctx context.Context, username, password string) (*LoginResult, error) {
	pp := l.page.Context(ctx)

	// 访问BOSS直聘登录页
	pp.MustNavigate("https://www.zhipin.com/user/login.html").MustWaitLoad()
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
