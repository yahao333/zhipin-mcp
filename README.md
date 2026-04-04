# Zhipin MCP 服务

BOSS直聘自动求职 MCP 服务，基于 Go + go-rod + MCP 协议，实现浏览器自动化求职操作。

## 核心能力

- **浏览器自动化**：go-rod 驱动 Chrome/Chromium，自动登录、搜索、投递、聊天
- **Cookie 持久化**：登录状态本地存储，重启无需重复扫码
- **投递去重**：自动检测职位是否已投递，避免重复
- **每日限额**：默认每日最多投递 30 封（可配置）
- **随机延时**：3-8 秒随机延时，模拟真实用户行为
- **定时任务**：Cron 表达式驱动，自动搜索并投递
- **AES 加密**：密码等敏感信息 AES 加密存储
- **SQLite 存储**：投递记录、配置信息本地持久化

## 快速开始

### 启动服务

```bash
# 无头模式运行
make run

# 调试模式（非无头，显示浏览器）
make run-debug

# 指定端口
make run-port
```

### 首次登录

```bash
# 获取登录二维码，浏览器窗口扫码
make api-login-qrcode-br
```

### 常用操作

```bash
# 搜索职位
make api-search

# 投递简历
make api-deliver ID=job123456

# 已投递列表
make api-delivered

# 投递统计
make api-stats

# 发送消息
make api-send-message
```

## 配置

配置文件位于 `configs/config.yaml`，支持以下配置项：

```yaml
mcp:
  host: "0.0.0.0"
  port: 18061

browser:
  headless: true
  user_data_dir: "./data/browser"

app:
  max_daily: 30      # 每日投递上限
  delay_min: 3        # 随机延时最小值（秒）
  delay_max: 8        # 随机延时最大值（秒）

security:
  aes_key: ""         # AES 加密密钥（32字符）
```

## 技术栈

- **语言**：Go 1.25.0
- **浏览器自动化**：go-rod
- **HTTP 框架**：Gin
- **数据库**：SQLite
- **日志**：logrus
- **协议**：MCP（Model Context Protocol）

## 交流群

联系方式：[微信](https://my.feishu.cn/docx/MeyzdbJc7o1P4DxGqRAcDlYEnOk?from=from_copylink)
