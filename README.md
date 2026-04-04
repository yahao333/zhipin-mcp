# Zhipin MCP 服务

BOSS直聘自动求职 MCP 服务，基于 Go + go-rod + MCP 协议，实现浏览器自动化求职操作。

## 功能列表

### 已实现

#### MCP 工具（16个）

| 工具 | 说明 |
|------|------|
| `check_login_status` | 检查登录状态 |
| `get_login_qrcode` | 获取登录二维码（Base64） |
| `delete_cookies` | 删除 Cookie 并重置登录状态 |
| `search_jobs` | 搜索职位（支持城市、薪资、经验、学历等筛选） |
| `get_job_detail` | 获取职位详情 |
| `deliver_job` | 投递简历到指定职位 |
| `delivered_list` | 获取已投递职位列表 |
| `batch_deliver` | 批量投递简历 |
| `start_cron` | 启动定时自动求职任务 |
| `stop_cron` | 停止定时任务 |
| `get_config` | 获取当前配置 |
| `update_config` | 更新配置（用户名、每日投递上限等） |
| `get_stats` | 获取投递统计（今日/累计） |
| `list_messages` | 获取消息列表 |
| `delete_message` | 删除消息（支持批量） |
| `send_message` | 向 HR 发送消息 |

#### REST API（18个端点）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/login/status` | 登录状态 |
| GET | `/api/login/qrcode` | 登录二维码（Base64） |
| GET | `/api/login/qrcode/browser` | 扫码登录（浏览器窗口） |
| DELETE | `/api/login/cookies` | 删除 Cookie |
| POST | `/api/jobs/search` | 搜索职位 |
| GET | `/api/jobs/:job_id` | 职位详情 |
| POST | `/api/deliver` | 投递简历 |
| GET | `/api/delivered` | 已投递列表 |
| POST | `/api/batch/deliver` | 批量投递 |
| GET | `/api/stats` | 投递统计 |
| GET | `/api/config` | 获取配置 |
| PUT | `/api/config` | 更新配置 |
| POST | `/api/cron/start` | 启动定时任务 |
| POST | `/api/cron/stop` | 停止定时任务 |
| GET | `/api/messages` | 消息列表 |
| POST | `/api/messages/delete` | 删除消息 |
| POST | `/api/messages/send` | 发送消息 |

#### 核心能力

- **浏览器自动化**：go-rod 驱动 Chrome/Chromium，自动登录、搜索、投递、聊天
- **Cookie 持久化**：登录状态本地存储，重启无需重复扫码
- **投递去重**：自动检测职位是否已投递，避免重复
- **每日限额**：默认每日最多投递 30 封（可配置）
- **随机延时**：3-8 秒随机延时，模拟真实用户行为
- **定时任务**：Cron 表达式驱动，自动搜索并投递
- **AES 加密**：密码等敏感信息 AES 加密存储
- **SQLite 存储**：投递记录、配置信息本地持久化

### 待实现

| 功能 | 优先级 | 说明 |
|------|--------|------|
| `reply_message` 回复消息 | P1 | 对消息列表中的消息进行回复 |
| 简历管理（上传/更新简历） | P1 | 上传或更新简历文件 |
| 消息已读标记 | P2 | 将消息标记为已读 |
| `list_crons` 定时任务列表 | P2 | 查看当前所有定时任务 |
| 消息搜索 | P2 | 在消息列表中搜索特定内容 |
| 附件发送 | P2 | 支持发送附件（如简历、图片） |
| 投递结果主动通知 | P2 | 定时任务投递后通过 Webhook 通知 |
| 多账号支持 | P3 | 支持同时管理多个 BOSS 直聘账号 |
| 简历解析与匹配 | P3 | 分析简历与职位JD的匹配度 |

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
# 获取登录二维码
make api-login-qrcode

# 或浏览器窗口扫码
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
