# Core Profile

The profile package manages user and group profiles for multi-user identity recognition. It enables the bot to identify users across different messaging platforms and associate memories with specific people or groups.

## Overview

When the bot interacts with multiple users across different platforms (Feishu, Telegram, Discord, etc.), the profile system helps:

- **Identity recognition** — Map platform-specific user IDs to consistent internal IDs
- **Memory association** — Link memories to specific users or groups
- **Multi-platform support** — Single profile can reference multiple platform IDs

## File Format

Profiles are stored in `PROFILES.md`:

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

## Data Structure

### Profile

```go
type Profile struct {
    ID          string            `json:"id"`           // Unique identifier
    Type        ProfileType       `json:"type"`         // "user" or "group"
    DisplayName string            `json:"display_name"` // Human-readable name
    PlatformIDs map[string]string `json:"platform_ids"` // Platform-specific IDs
    Attributes  map[string]string `json:"attributes"`   // Custom attributes
}

type ProfileType string

const (
    TypeUser  ProfileType = "user"
    TypeGroup ProfileType = "group"
)
```

### Supported Platforms

The following platform IDs are automatically recognized:

- `feishu` — Lark/Feishu user ID
- `telegram` — Telegram username or ID
- `discord` — Discord user ID
- `slack` — Slack user ID
- `dingtalk` — DingTalk user ID
- `wecom` — WeCom/WeChat Work user ID
- `qq` — QQ user ID
- `line` — LINE user ID

Other attributes are stored in the `Attributes` map.

## Usage

### Creating a Profile Manager

```go
import "github.com/cc-connect-memory/core/profile"

// Create manager with data directory
mgr := profile.NewManager("/data/bot/myproject")
```

### Loading Profiles

```go
// Load from PROFILES.md
if err := mgr.Load(); err != nil {
    log.Fatal(err)
}

// Or create default file if it doesn't exist
created, err := profile.EnsureDefaultFile("/data/bot/myproject")
if err != nil {
    log.Fatal(err)
}
if created {
    fmt.Println("Created default PROFILES.md")
}
```

### Querying Profiles

```go
// Get by internal ID
p := mgr.Get("user_001")
if p != nil {
    fmt.Printf("Name: %s, Type: %s\n", p.DisplayName, p.Type)
}

// Find by platform ID (e.g., from incoming message)
p := mgr.GetByPlatformID("telegram", "@zhangsan")
if p != nil {
    fmt.Printf("Found user: %s\n", p.DisplayName)
}

// Get all profiles
all := mgr.GetAll()

// Filter by type
users := mgr.GetByType(profile.TypeUser)
groups := mgr.GetByType(profile.TypeGroup)
```

### Modifying Profiles

```go
// Add new profile
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

// Update existing profile
p := mgr.Get("user_001")
p.Attributes["Note"] = "VIP customer"
if err := mgr.Update(p); err != nil {
    log.Error(err)
}

// Delete profile
if err := mgr.Delete("user_001"); err != nil {
    log.Error(err)
}

// Save changes to file
if err := mgr.Save(); err != nil {
    log.Error(err)
}
```

## Integration with Memory

The profile system works with the memory system to attribute memories to specific users:

```go
import "github.com/cc-connect-memory/core/memory"

// When processing a message
userID := extractUserID(msg)           // e.g., "user_001"
platformID := extractPlatformID(msg)  // e.g., "@zhangsan"
platform := "telegram"

// Look up profile
profileMgr := getProfileManager()
if profile := profileMgr.GetByPlatformID(platform, platformID); profile != nil {
    userID = profile.ID
}

// Build metadata for memory
metadata := memory.BuildProfileMetadata(
    userID,           // "user_001"
    "",               // channel identity ID
    profile.DisplayName, // "张三"
    platform,         // "telegram"
)

// Store memory with profile info
store.Add(ctx, memory.AddRequest{
    Memory:   "User prefers dark mode",
    Metadata: metadata,
})
```

## Context in Memory

When memories include profile metadata, they can be filtered:

```go
// Search memories for specific user
results, err := store.Search(ctx, memory.SearchRequest{
    Query: "preferences",
    Filters: map[string]any{
        memory.MetadataKeyUserID: "user_001",
    },
})

// Search memories for specific channel
results, err := store.Search(ctx, memory.SearchRequest{
    Query: "project",
    Filters: map[string]any{
        memory.MetadataKeyChannelIdentity: "channel_abc",
    },
})
```

## Thread Safety

The Manager type is thread-safe. All methods that access or modify profiles use `sync.RWMutex` for concurrent read access with exclusive write access.

## Configuration

### Directory Structure

```
{data_dir}/
  bot/{project}/
    PROFILES.md    # User and group profiles
```

The project name comes from the configuration, allowing multiple profile sets.

### Default File Template

When `EnsureDefaultFile()` creates a new profile file:

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

## Best Practices

### Profile Lookup Performance

For high-traffic platforms, cache the profile manager:

```go
// At startup
profileMgr := profile.NewManager(dataDir)
if err := profileMgr.Load(); err != nil {
    log.Warn("failed to load profiles", "error", err)
}

// Store in engine
engine.SetProfileManager(profileMgr)
```

### Handling New Users

Automatically create profiles for new users:

```go
func ensureProfile(mgr *profile.Manager, platform, platformID, displayName string) *profile.Profile {
    // Check existing
    if p := mgr.GetByPlatformID(platform, platformID); p != nil {
        return p
    }

    // Create new profile
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
        log.Error("failed to add profile", "error", err)
        return nil
    }

    return p
}
```

## Integration

The profile manager is injected into the Engine via `core.SetProfileManager()`. The Engine uses it to:

1. Identify users from incoming messages
2. Associate memories with user profiles
3. Provide user context to the agent

See [CLAUDE.md](../CLAUDE.md) for integration details.
