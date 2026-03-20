package main

import (
	"context"
	"flag"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/configs"
)

func main() {
	var (
		headless bool
		binPath  string // 浏览器二进制文件路径
		port     string
		mode     string // 运行模式: http 或 stdio
	)
	flag.StringVar(&mode, "mode", "stdio", "运行模式: http 或 stdio")
	flag.BoolVar(&headless, "headless", true, "是否无头模式")
	flag.StringVar(&binPath, "bin", "", "浏览器二进制文件路径")
	flag.StringVar(&port, "port", ":18061", "端口")
	flag.Parse()

	// 初始化配置
	if err := configs.Init(); err != nil {
		logrus.Warnf("初始化配置失败: %v", err)
	}

	// 设置命令行参数覆盖配置
	if len(binPath) == 0 {
		binPath = os.Getenv("ROD_BROWSER_BIN")
	}

	configs.InitHeadless(headless)
	configs.SetBinPath(binPath)

	// 设置日志
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 初始化服务
	zhipinService := NewZhipinService()

	// 根据模式启动
	if mode == "stdio" {
		// STDIO 模式
		logrus.Info("启动 MCP STDIO 模式...")
		mcpServer := NewMCPServer(zhipinService)
		stdioServer := NewMCPStdioServer(mcpServer)
		if err := stdioServer.Run(context.Background()); err != nil {
			logrus.Fatalf("STDIO 服务器错误: %v", err)
		}
	} else {
		// HTTP 模式
		logrus.Info("启动 MCP HTTP 模式...")
		appServer := NewAppServer(zhipinService)
		if err := appServer.Start(port); err != nil {
			logrus.Fatalf("启动服务器失败: %v", err)
		}
	}
}
