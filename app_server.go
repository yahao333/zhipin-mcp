package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/configs"
	"github.com/yahao333/zhipin-mcp/zhipin"
)

// AppServer 应用服务器
type AppServer struct {
	server        *http.Server
	zhipinService *ZhipinService
	mcpServer     *MCPServer
	wg            sync.WaitGroup
}

// NewAppServer 创建应用服务器
func NewAppServer(service *ZhipinService) *AppServer {
	app := &AppServer{
		zhipinService: service,
		mcpServer:     NewMCPServer(service),
	}

	// 设置路由
	router := setupRoutes(app)

	app.server = &http.Server{
		Addr:    configs.Port,
		Handler: router,
	}

	return app
}

// Start 启动应用服务器
func (s *AppServer) Start(port string) error {
	// 如果指定了端口，覆盖配置
	if port != "" {
		s.server.Addr = port
	}

	// 初始化数据库
	if err := initDatabase(); err != nil {
		logrus.Warnf("初始化数据库失败: %v", err)
	}

	// 启动定时任务管理器
	zhipin.StartCron()

	logrus.Infof("服务器启动: %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop 停止应用服务器
func (s *AppServer) Stop() error {
	ctx := s.server.Shutdown(context.Background())
	return ctx
}

// Wait 等待服务器关闭
func (s *AppServer) Wait() {
	s.wg.Add(1)
	s.wg.Wait()
}
