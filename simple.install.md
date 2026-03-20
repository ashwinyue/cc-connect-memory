# 快速安装启动指南

## 步骤 1: 复制配置文件

```bash
cp config.example.toml config.toml
```

## 步骤 2: 配置飞书机器人

### 方式 A: 使用 CLI 自动配置（推荐）

```bash
# 创建新应用并扫码绑定
make feishu-setup PROJECT=fish

# 或绑定已有应用
make feishu-setup PROJECT=fish APP_ID=cli_xxx:sec_xxx
```

### 方式 B: 手动配置

1. 前往 https://open.feishu.cn 创建企业应用
2. 启用机器人能力，添加权限 `im:message.receive_v1`, `im:message:send_as_bot`
3. 配置事件订阅，选择 WebSocket 长连接模式，添加事件 `im.message.receive_v1`
4. 发布应用，获取 App ID 和 App Secret
5. 将凭证填入 config.toml：

```toml
[[projects]]
name = "fish"

[projects.agent]
type = "claudecode"
[projects.agent.options]
work_dir = "/path/to/your/project"
mode = "default"

[[projects.platforms]]
type = "feishu"
[projects.platforms.options]
app_id = "cli_xxx"
app_secret = "xxx"
```

## 步骤 3: 配置记忆功能（可选）

在 config.toml 中添加 `[memory]` 配置，使用 Kimi API 提取对话事实：

```toml
[memory]
enabled = true
provider = "anthropic"
api_key = "sk-kimi-xxx"
base_url = "https://api.kimi.com/coding"
model = "kimi"
max_items = 8
```

支持的 provider:
- `anthropic`: Anthropic API (含兼容端点如 Kimi、DeepSeek)
- 可通过 `base_url` 自定义端点

## 步骤 4: 编译安装

```bash
make build
make install
```

## 步骤 5: 启动服务

```bash
make start
```

## 常用命令

```bash
make start    # 启动
make stop     # 停止
make restart  # 重启
make logs     # 查看日志
```

## 注意事项

- 首次运行需确保飞书应用已发布并配置好可用范围
- 记忆功能需要配置有效的 LLM API Key
- 工作目录 `work_dir` 需要是绝对路径
