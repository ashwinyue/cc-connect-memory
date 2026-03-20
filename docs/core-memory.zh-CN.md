# 核心记忆系统

记忆包为 AI 助手提供跨会话记忆功能。它使机器人能够记住之前会话中的信息，并在响应前检索相关的上下文。

## 概述

记忆系统采用简单的基于文件的方法，无需向量数据库依赖。记忆存储为按日期组织的 Markdown 文件，并可选地使用基于 LLM 的提取器从会话中自动捕获事实。

主要功能：
- **自动记忆提取** — LLM 从会话历史中提取事实
- **基于文本的搜索** — 简单的子字符串匹配，无需嵌入
- **冲突解决** — 内置去重和合并策略
- **记忆压缩** — 当记忆增长过多时自动清理
- **CRUD 操作** — 完全控制记忆条目

## 架构

```
memory/
├── provider.go    # Provider 接口
├── types.go       # 数据类型和常量
├── storefs.go     # 基于文件的存储实现
├── extractor.go   # 基于 LLM 的事实提取
├── decide.go      # 记忆冲突解决
├── compact.go     # 记忆压缩
└── requests.go    # 请求/响应类型
```

### Provider 接口

所有记忆实现必须满足的核心接口：

```go
type Provider interface {
    // 会话前：检索相关记忆
    OnBeforeChat(ctx context.Context, req BeforeChatRequest) (*BeforeChatResult, error)

    // 会话后：存储新记忆
    OnAfterChat(ctx context.Context, req AfterChatRequest) error

    // 搜索记忆
    Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)

    // 添加新记忆
    Add(ctx context.Context, req AddRequest) (*MemoryItem, error)

    // 获取所有记忆
    GetAll(ctx context.Context, filters map[string]any) ([]MemoryItem, error)

    // 更新记忆
    Update(ctx context.Context, id string, memory string) (*MemoryItem, error)

    // 删除记忆
    Delete(ctx context.Context, id string) error

    // 批量删除
    DeleteBatch(ctx context.Context, ids []string) error

    // 删除全部
    DeleteAll(ctx context.Context, filters map[string]any) error

    // 获取使用统计
    Usage(ctx context.Context, filters map[string]any) (*UsageResponse, error)

    // 压缩记忆
    Compact(ctx context.Context) (*CompactResult, error)
}
```

## 文件格式

### 每日记忆文件

记忆存储在 `{data_dir}/memory/YYYY-MM-DD.md`：

```markdown
# Memory 2026-03-20

## Entry mem_1234567890

```yaml
id: mem_1234567890
hash: a1b2c3d4
created_at: 2026-03-20T10:30:00Z
updated_at: 2026-03-20T10:30:00Z
metadata:
  profile_platform: telegram
  profile_user_id: user_123
```

用户更喜欢深色模式进行开发。

## Entry mem_1234567891

```yaml
id: mem_1234567891
hash: e5f6g7h8
created_at: 2026-03-20T14:22:00Z
metadata:
  profile_platform: feishu
  profile_channel_identity_id: channel_abc
```

用户正在开发一个名为 cc-connect 的 Go 项目。
```

### 记忆概览

摘要文件位于 `{data_dir}/MEMORY.md`：

```markdown
# MEMORY

_这是你的核心记忆，请保持更新。_

1. [2026-03-20] 用户更喜欢深色模式进行开发。
2. [2026-03-20] 用户正在开发一个名为 cc-connect 的 Go 项目。
3. [2026-03-19] 用户名叫张三。
```

## 使用方法

### 创建记忆存储

```go
import "github.com/cc-connect-memory/core/memory"

// 创建基于文件的存储
store := memory.NewStoreFS("/data", extractor, logger)

// 配置可选组件
store.SetDecider(memory.NewSimpleDecider())
store.SetCompactor(memory.NewSimpleCompactor())
```

### 会话前 — 检索上下文

```go
result, err := store.OnBeforeChat(ctx, memory.BeforeChatRequest{
    Query:     "如何配置深色模式？",
    SessionID: "session_123",
    Platform:  "telegram",
    UserID:    "user_456",
    ChatID:    "chat_789",
})
// 返回类似：
// <memory-context>
// 之前会话的相关上下文：
// - 用户更喜欢深色模式进行开发。
// </memory-context>
```

### 会话后 — 存储记忆

```go
err := store.OnAfterChat(ctx, memory.AfterChatRequest{
    SessionID: "session_123",
    Platform:  "telegram",
    UserID:    "user_456",
    Messages: []memory.MemoryMessage{
        {Role: "user", Content: "我正在开发一个新的 Go 项目。"},
        {Role: "assistant", Content: "太棒了！叫什么名字？"},
        {Role: "user", Content: "叫 cc-connect-memory。"},
    },
    AgentReply: "好的，我会记住它。",
})
// 提取器会提取 "用户正在开发一个名为 cc-connect-memory 的项目"
// 并自动存储。
```

### 手动记忆操作

```go
// 手动添加记忆
item, err := store.Add(ctx, memory.AddRequest{
    Memory: "用户更喜欢 VS Code 而不是 IntelliJ",
    Metadata: map[string]any{
        "platform": "telegram",
        "user_id":  "user_456",
    },
})

// 搜索记忆
results, err := store.Search(ctx, memory.SearchRequest{
    Query: "偏好",
    Limit: 5,
})

// 更新记忆
updated, err := store.Update(ctx, "mem_123", "更新后的记忆内容")

// 删除记忆
err := store.Delete(ctx, "mem_123")

// 批量删除记忆
err := store.DeleteBatch(ctx, []string{"mem_123", "mem_456"})

// 删除所有记忆
err := store.DeleteAll(ctx, nil)

// 获取使用统计
usage, err := store.Usage(ctx, nil)
// 返回: {Count: 42, TotalBytes: 12345, AvgItemBytes: 294}

// 压缩记忆（合并/删除旧条目）
result, err := store.Compact(ctx)
```

### 禁用记忆

不需要记忆时使用 `NoopProvider`：

```go
var provider memory.Provider = &memory.NoopProvider{}
```

## 提取器

提取器接口允许自定义事实提取逻辑：

```go
type Extractor interface {
    Extract(ctx context.Context, messages []MemoryMessage) ([]string, error)
}
```

基于 LLM 的提取器分析会话消息并返回要存储的事实陈述。

## 冲突解决

`Decider` 接口处理记忆冲突：

```go
type Decider interface {
    Decide(ctx context.Context, newMemory string, existing []MemoryItem) (Decision, error)
}

type Decision struct {
    Type      DecisionType  // DecisionAdd, DecisionUpdate, DecisionNoop
    MemoryID  string        // 要更新的记忆 ID（用于 DecisionUpdate）
    Reason    string        // 决策说明
}
```

内置选项：
- `SimpleDecider` — 通过哈希检查完全重复，如有则跳过

## 压缩

`Compactor` 接口管理记忆清理：

```go
type Compactor interface {
    Compact(ctx context.Context, items []MemoryItem, targetCount int) ([]MemoryItem, error)
}
```

内置选项：
- `SimpleCompactor` — 删除最旧的记忆以达到目标数量

当记忆数量超过 `MaxMemoryItems`（默认：1000）时触发自动压缩。

## 配置

### 记忆系统配置

在 `config.toml` 中：

```toml
[memory]
enabled   = true
provider  = "anthropic"     # 用于事实提取的 LLM
api_key   = "sk-ant-..."    # API 密钥
model     = "claude-3-5-haiku-latest"
max_items = 8                # 每次检索的最大记忆数
```

### 目录结构

```
{data_dir}/
  memory/
    2026-03-20.md    # 每日记忆文件
    2026-03-19.md
    2026-03-18.md
    ...
  MEMORY.md         # 概览文件
```

## 集成

记忆提供者通过 `core.SetMemoryProvider()` 注入到引擎中。引擎自动：

1. 在每个用户消息之前调用 `OnBeforeChat()`
2. 在每个智能体响应之后调用 `OnAfterChat()`

有关集成详情，请参阅 [CLAUDE.md](../CLAUDE.md)。

## 常量

```go
const (
    DefaultMemoryLimit    = 8    // 检索的最大记忆数
    MaxMemoryContextItems = 16   # 上下文中的最大条目数
    MemoryContextMaxChars = 220  # 每个记忆的最大字符数
)
```
