package browser

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/zhipin-mcp/cookies"
	"github.com/xpzouying/zhipin-mcp/configs"
)

// ErrCookiesNotFound cookies 文件不存在的错误
var ErrCookiesNotFound = errors.New("cookies 文件不存在，请先调用 /api/login/qrcode 获取登录二维码并扫码登录")

type browserConfig struct {
	binPath string
}

type Option func(*browserConfig)

func WithBinPath(binPath string) Option {
	return func(c *browserConfig) {
		c.binPath = binPath
	}
}

// NewBrowser 创建浏览器实例
func NewBrowser(headless bool, options ...Option) (*headless_browser.Browser, error) {
	cfg := &browserConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	opts := []headless_browser.Option{
		headless_browser.WithHeadless(headless),
	}
	if cfg.binPath != "" {
		opts = append(opts, headless_browser.WithChromeBinPath(cfg.binPath))
	}

	// 加载 cookies
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)
	data, err := cookieLoader.LoadCookies()
	if err != nil {
		if cookies.IsCookieNotFound(err) {
			logrus.Warnf("cookies 文件不存在: %v", err)
			return nil, ErrCookiesNotFound
		}
		logrus.Warnf("failed to load cookies: %v", err)
		return nil, err
	}

	opts = append(opts, headless_browser.WithCookies(string(data)))
	logrus.Debugf("loaded cookies from file successfully")

	return headless_browser.New(opts...), nil
}

// CloseBrowser 关闭浏览器
func CloseBrowser(b *headless_browser.Browser) {
	if b != nil {
		logrus.Info("浏览器已关闭")
	}
}

// SetupBrowser 创建并配置浏览器
func SetupBrowser() (*headless_browser.Browser, error) {
	return NewBrowser(configs.IsHeadless(), WithBinPath(configs.GetBinPath()))
}
