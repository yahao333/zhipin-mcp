package zhipin

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	// 访问BOSS直聘首页
	err := l.page.Navigate("https://www.zhipin.com/")
	if err != nil {
		return false, errors.Wrap(err, "访问首页失败")
	}

	// 等待页面加载
	l.page.WaitLoad()

	// 等待页面稳定
	time.Sleep(2 * time.Second)

	html, err := l.page.HTML()
	if err != nil {
		return false, err
	}

	// 检查登录状态
	// 已登录：页面包含用户头像或用户名
	// 未登录：页面包含登录/注册按钮
	isLoggedIn := strings.Contains(html, "user-name") ||
		strings.Contains(html, "nick-name") ||
		strings.Contains(html, "boss-avatar") ||
		strings.Contains(html, "头像")

	// 如果不包含登录按钮，认为已登录
	if !strings.Contains(html, "btn-login") && !strings.Contains(html, "登录/注册") {
		isLoggedIn = true
	}

	return isLoggedIn, nil
}

// FetchQrcodeImage 获取登录二维码
func (l *Login) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	// 访问BOSS直聘登录页
	err := l.page.Navigate("https://www.zhipin.com/user/login.html")
	if err != nil {
		return "", false, errors.Wrap(err, "访问登录页失败")
	}

	// 等待页面加载
	l.page.WaitLoad()

	// 等待二维码加载
	time.Sleep(2 * time.Second)

	// 查找二维码图片 - BOSS直聘的实际选择器
	img, err := l.page.Element(".qrcode img")
	if err != nil {
		// 尝试其他选择器
		img, err = l.page.Element(".login-qrcode img")
		if err != nil {
			img, err = l.page.Element("#qrcode")
			if err != nil {
				return "", false, errors.Wrap(err, "查找二维码失败")
			}
		}
	}

	// 获取二维码图片的Base64
	src, err := img.Attribute("src")
	if err != nil {
		return "", false, errors.Wrap(err, "获取二维码URL失败")
	}

	var imgBase64 string
	if strings.HasPrefix(*src, "data:image") {
		imgBase64 = *src
	} else if strings.HasPrefix(*src, "//") {
		imgBase64 = "https:" + *src
	} else {
		imgBase64 = *src
	}

	// 检查是否已登录（可能是扫码状态）
	isLoggedIn, _ := l.CheckLoginStatus(ctx)

	return imgBase64, isLoggedIn, nil
}

// WaitForLogin 等待扫码登录成功
func (l *Login) WaitForLogin(ctx context.Context) bool {
	// 每3秒检查一次登录状态，最多等待4分钟
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	timeout := time.After(4 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return false
		case <-timeout:
			logrus.Warn("登录超时")
			return false
		case <-ticker.C:
			isLoggedIn, err := l.CheckLoginStatus(ctx)
			if err != nil {
				logrus.Warnf("检查登录状态失败: %v", err)
				continue
			}
			if isLoggedIn {
				logrus.Info("登录成功")
				return true
			}
		}
	}
}

// LoginWithPassword 使用密码登录
func (l *Login) LoginWithPassword(ctx context.Context, username, password string) (*LoginResult, error) {
	// 访问BOSS直聘登录页
	err := l.page.Navigate("https://www.zhipin.com/user/login.html")
	if err != nil {
		return nil, errors.Wrap(err, "访问登录页失败")
	}

	l.page.WaitLoad()
	time.Sleep(2 * time.Second)

	// 切换到密码登录（如果需要）
	l.page.MustElement("a[tab='account']").MustClick()
	time.Sleep(1 * time.Second)

	// 输入用户名
	l.page.MustElement("#account").MustInput(username)

	// 输入密码
	l.page.MustElement("#password").MustInput(password)

	// 点击登录按钮
	l.page.MustElement(".btn-login").MustClick()

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
	html, _ := l.page.HTML()
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
