# 核心档案系统

档案包管理用户和群组档案，用于多用户身份识别。它使机器人能够识别来自不同消息平台的用户，并将记忆与特定的人或群组关联起来。

## 概述

当机器人在不同平台（飞书、Telegram、Discord 等）与多个用户交互时，档案系统有助于：

- **身份识别** — 将平台特定的用户 ID 映射到一致的内部 ID
- **记忆关联** — 将记忆链接到特定用户或群组
- **多平台支持** — 单个档案可以引用多个平台 ID

## 文件格式

档案存储在 `PROFILES.md`：

```markdown
_This is profiles from different users or groups._

## 张三
- ID: user_001
- Type: user
- Email: zhangsan@example.com
- Telegram: @zhangsan
- Feishu: ou_xxx

## 李四
- ID: user_002
- Type: user
- Discord: user_456
- Slack: @lisi

## 开发组
- ID: group_dev
- Type: group
- Members: 张三, 李四, 王五
- Platform: feishu
- Channel: 开发组频道
```

## 数据结构

### 档案

```go
type Profile struct {
    ID          string            `json:"id"`           // 唯一标识符
    Type        ProfileType       `json:"type"`         // "user" 或 "group"
    DisplayName string            `json:"display_name"` // 人类可读的名称
    PlatformIDs map[string]string `json:"platform_ids"` // 平台特定的 ID
    Attributes  map[string]string `json:"attributes"`   // 自定义属性
}

type ProfileType string

const (
    TypeUser  ProfileType = "user"
    TypeGroup ProfileType = "group"
)
```

### 支持的平台

以下平台 ID 自动识别：

- `feishu` — 飞书用户 ID
- `telegram` — Telegram 用户名或 ID
- `discord` — Discord 用户 ID
- `slack` — Slack 用户 ID
- `dingtalk` — 钉钉用户 ID
- `wecom` — 企业微信用户 ID
- `qq` — QQ 用户 ID
- `line` — LINE 用户 ID

其他属性存储在 `Attributes` 映射中。

## 使用方法

### 创建档案管理器

```go
import "github.com/cc-connect-memory/core/profile"

// 使用数据目录创建管理器
mgr := profile.NewManager("/data/bot/myproject")
```

### 加载档案

```go
// 从 PROFILES.md 加载
if err := mgr.Load(); err != nil {
    log.Fatal(err)
}

// 或者如果不存在则创建默认文件
created, err := profile.EnsureDefaultFile("/data/bot/myproject")
if err != nil {
    log.Fatal(err)
}
if created {
    fmt.Println("已创建默认 PROFILES.md")
}
```

### 查询档案

```go
// 通过内部 ID 获取
p := mgr.Get("user_001")
if p != nil {
    fmt.Printf("名称: %s, 类型: %s\n", p.DisplayName, p.Type)
}

// 通过平台 ID 查找（例如来自传入消息）
p := mgr.GetByPlatformID("telegram", "@zhangsan")
if p != nil {
    fmt.Printf("找到用户: %s\n", p.DisplayName)
}

// 获取所有档案
all := mgr.GetAll()

// 按类型筛选
users := mgr.GetByType(profile.TypeUser)
groups := mgr.GetByType(profile.TypeGroup)
```

### 修改档案

```go
// 添加新档案
newProfile := &profile.Profile{
    ID:          "user_003",
    Type:        profile.TypeUser,
    DisplayName: "王五",
    PlatformIDs: map[string]string{
        "telegram": "@wangwu",
    },
    Attributes: map[string]string{
        "Email": "wangwu@example.com",
    },
}
if err := mgr.Add(newProfile); err != nil {
    log.Error(err)
}

// 更新现有档案
p := mgr.Get("user_001")
p.Attributes["Note"] = "VIP 客户"
if err := mgr.Update(p); err != nil {
    log.Error(err)
}

// 删除档案
if err := mgr.Delete("user_001"); err != nil {
    log.Error(err)
}

// 保存更改到文件
if err := mgr.Save(); err != nil {
    log.Error(err)
}
```

## 与记忆系统集成

档案系统与记忆系统协同工作，将记忆归因于特定用户：

```go
import "github.com/cc-connect-memory/core/memory"

// 处理消息时
userID := extractUserID(msg)           // 例如 "user_001"
platformID := extractPlatformID(msg)  // 例如 "@zhangsan"
platform := "telegram"

// 查找档案
profileMgr := getProfileManager()
if profile := profileMgr.GetByPlatformID(platform, platformID); profile != nil {
    userID = profile.ID
}

// 为记忆构建元数据
metadata := memory.BuildProfileMetadata(
    userID,           // "user_001"
    "",               // 频道身份 ID
    profile.DisplayName, // "张三"
    platform,         // "telegram"
)

// 存储带档案信息的记忆
store.Add(ctx, memory.AddRequest{
    Memory:   "用户更喜欢深色模式",
    Metadata: metadata,
})
```

### 记忆中的上下文

当记忆包含档案元数据时，可以进行筛选：

```go
// 搜索特定用户的记忆
results, err := store.Search(ctx, memory.SearchRequest{
    Query: "偏好",
    Filters: map[string]any{
        memory.MetadataKeyUserID: "user_001",
    },
})

// 搜索特定频道的记忆
results, err := store.Search(ctx, memory.SearchRequest{
    Query: "项目",
    Filters: map[string]any{
        memory.MetadataKeyChannelIdentity: "channel_abc",
    },
})
```

## 线程安全

Manager 类型是线程安全的。所有访问或修改档案的方法都使用 `sync.RWMutex` 来提供并发读取和独占写入访问。

## 配置

### 目录结构

```
{data_dir}/
  bot/{project}/
    PROFILES.md    # 用户和群组档案
```

项目名称来自配置，允许多个档案集。

### 默认文件模板

当 `EnsureDefaultFile()` 创建新档案文件时：

```markdown
_This is profiles from different users or groups._

## 示例用户
- ID: user_example
- Type: user
- Email: example@example.com
- Telegram: @example_user
- Feishu: ou_xxx

## 示例群组
- ID: group_example
- Type: group
- Members: 示例用户, 其他成员
- Platform: feishu
- Channel: 示例群组频道
```

## 最佳实践

### 档案查询性能

对于高流量平台，缓存档案管理器：

```go
// 启动时
profileMgr := profile.NewManager(dataDir)
if err := profileMgr.Load(); err != nil {
    log.Warn("加载档案失败", "error", err)
}

// 存储在引擎中
engine.SetProfileManager(profileMgr)
```

### 处理新用户

为新用户自动创建档案：

```go
func ensureProfile(mgr *profile.Manager, platform, platformID, displayName string) *profile.Profile {
    // 检查现有
    if p := mgr.GetByPlatformID(platform, platformID); p != nil {
        return p
    }

    // 创建新档案
    newID := "user_" + generateID()
    p := &profile.Profile{
        ID:          newID,
        Type:        profile.TypeUser,
        DisplayName: displayName,
        PlatformIDs: map[string]string{
            platform: platformID,
        },
    }

    if err := mgr.Add(p); err != nil {
        log.Error("添加档案失败", "error", err)
        return nil
    }

    return p
}
```

## 集成

档案管理器通过 `core.SetProfileManager()` 注入到引擎中。引擎使用它来：

1. 从传入消息中识别用户
2. 将记忆与用户档案关联
3. 为智能体提供用户上下文

有关集成详情，请参阅 [CLAUDE.md](../CLAUDE.md)。
