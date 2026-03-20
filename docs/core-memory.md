# Core Memory

The memory package provides cross-session memory for the AI assistant. It enables the bot to remember information from previous conversations and retrieve relevant context before responding.

## Overview

The memory system uses a simple file-based approach without vector database dependencies. Memories are stored as Markdown files organized by date, with an optional LLM-based extractor for automatically capturing facts from conversations.

Key features:
- **Automatic memory extraction** — LLM extracts facts from conversation history
- **Text-based search** — Simple substring matching without embeddings
- **Conflict resolution** — Built-in deduplication and merge strategies
- **Memory compaction** — Automatic cleanup when memory grows too large
- **CRUD operations** — Full control over memory entries

## Architecture

```
memory/
├── provider.go    # Provider interface
├── types.go       # Data types and constants
├── storefs.go     # File-based storage implementation
├── extractor.go   # LLM-based fact extraction
├── decide.go      # Memory conflict resolution
├── compact.go     # Memory compaction
└── requests.go    # Request/response types
```

### Provider Interface

The core interface all memory implementations must satisfy:

```go
type Provider interface {
    // Before chat: retrieve relevant memories
    OnBeforeChat(ctx context.Context, req BeforeChatRequest) (*BeforeChatResult, error)

    // After chat: store new memories
    OnAfterChat(ctx context.Context, req AfterChatRequest) error

    // Search memories
    Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)

    // Add new memory
    Add(ctx context.Context, req AddRequest) (*MemoryItem, error)

    // Get all memories
    GetAll(ctx context.Context, filters map[string]any) ([]MemoryItem, error)

    // Update memory
    Update(ctx context.Context, id string, memory string) (*MemoryItem, error)

    // Delete memory
    Delete(ctx context.Context, id string) error

    // Delete batch
    DeleteBatch(ctx context.Context, ids []string) error

    // Delete all
    DeleteAll(ctx context.Context, filters map[string]any) error

    // Get usage stats
    Usage(ctx context.Context, filters map[string]any) (*UsageResponse, error)

    // Compact memories
    Compact(ctx context.Context) (*CompactResult, error)
}
```

## File Format

### Daily Memory Files

Memories are stored in `{data_dir}/memory/YYYY-MM-DD.md`:

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

User prefers dark mode for development.

## Entry mem_1234567891

```yaml
id: mem_1234567891
hash: e5f6g7h8
created_at: 2026-03-20T14:22:00Z
metadata:
  profile_platform: feishu
  profile_channel_identity_id: channel_abc
```

The user is working on a Go project called cc-connect.
```

### Memory Overview

A summary file at `{data_dir}/MEMORY.md`:

```markdown
# MEMORY

_This is your core memory, please keep it up to date._

1. [2026-03-20] User prefers dark mode for development.
2. [2026-03-20] The user is working on a Go project called cc-connect.
3. [2026-03-19] User's name is 张三.
```

## Usage

### Creating a Memory Store

```go
import "github.com/cc-connect-memory/core/memory"

// Create file-based store
store := memory.NewStoreFS("/data", extractor, logger)

// Configure optional components
store.SetDecider(memory.NewSimpleDecider())
store.SetCompactor(memory.NewSimpleCompactor())
```

### Before Chat — Retrieve Context

```go
result, err := store.OnBeforeChat(ctx, memory.BeforeChatRequest{
    Query:     "How do I configure dark mode?",
    SessionID: "session_123",
    Platform:  "telegram",
    UserID:    "user_456",
    ChatID:    "chat_789",
})
// Returns context like:
// <memory-context>
// Relevant context from previous conversations:
// - User prefers dark mode for development.
// </memory-context>
```

### After Chat — Store Memories

```go
err := store.OnAfterChat(ctx, memory.AfterChatRequest{
    SessionID: "session_123",
    Platform:  "telegram",
    UserID:    "user_456",
    Messages: []memory.MemoryMessage{
        {Role: "user", Content: "I'm working on a new Go project."},
        {Role: "assistant", Content: "That sounds great! What's it called?"},
        {Role: "user", Content: "It's called cc-connect-memory."},
    },
    AgentReply: "Great! I'll remember that.",
})
// The extractor will extract "The user is working on a project called cc-connect-memory"
// and store it automatically.
```

### Manual Memory Operations

```go
// Add a memory manually
item, err := store.Add(ctx, memory.AddRequest{
    Memory: "User prefers VS Code over IntelliJ",
    Metadata: map[string]any{
        "platform": "telegram",
        "user_id":  "user_456",
    },
})

// Search memories
results, err := store.Search(ctx, memory.SearchRequest{
    Query: "preferences",
    Limit: 5,
})

// Update a memory
updated, err := store.Update(ctx, "mem_123", "Updated memory content")

// Delete a memory
err := store.Delete(ctx, "mem_123")

// Delete multiple memories
err := store.DeleteBatch(ctx, []string{"mem_123", "mem_456"})

// Delete all memories
err := store.DeleteAll(ctx, nil)

// Get usage statistics
usage, err := store.Usage(ctx, nil)
// Returns: {Count: 42, TotalBytes: 12345, AvgItemBytes: 294}

// Compact memories (merge/remove old entries)
result, err := store.Compact(ctx)
```

### Disabling Memory

Use `NoopProvider` when memory is not needed:

```go
var provider memory.Provider = &memory.NoopProvider{}
```

## Extractor

The extractor interface allows custom fact extraction logic:

```go
type Extractor interface {
    Extract(ctx context.Context, messages []MemoryMessage) ([]string, error)
}
```

The LLM-based extractor analyzes conversation messages and returns factual statements to store.

## Conflict Resolution

The `Decider` interface handles memory conflicts:

```go
type Decider interface {
    Decide(ctx context.Context, newMemory string, existing []MemoryItem) (Decision, error)
}

type Decision struct {
    Type      DecisionType  // DecisionAdd, DecisionUpdate, DecisionNoop
    MemoryID  string        // ID of memory to update (for DecisionUpdate)
    Reason    string        // Explanation for the decision
}
```

Built-in options:
- `SimpleDecider` — Checks for exact duplicates via hash, skips if found

## Compaction

The `Compactor` interface manages memory cleanup:

```go
type Compactor interface {
    Compact(ctx context.Context, items []MemoryItem, targetCount int) ([]MemoryItem, error)
}
```

Built-in options:
- `SimpleCompactor` — Removes oldest memories to reach target count

Auto-compaction triggers when memory count exceeds `MaxMemoryItems` (default: 1000).

## Configuration

### Memory System Config

In `config.toml`:

```toml
[memory]
enabled   = true
provider  = "anthropic"     # LLM for fact extraction
api_key   = "sk-ant-..."    # API key
model     = "claude-3-5-haiku-latest"
max_items = 8                # Max memories to retrieve per turn
```

### Directory Structure

```
{data_dir}/
  memory/
    2026-03-20.md    # Daily memory files
    2026-03-19.md
    2026-03-18.md
    ...
  MEMORY.md         # Overview file
```

## Integration

The memory provider is injected into the Engine via `core.SetMemoryProvider()`. The Engine automatically:

1. Calls `OnBeforeChat()` before each user message
2. Calls `OnAfterChat()` after each agent response

See [CLAUDE.md](../CLAUDE.md) for integration details.

## Constants

```go
const (
    DefaultMemoryLimit    = 8    // Max memories to retrieve
    MaxMemoryContextItems = 16   // Max items in context
    MemoryContextMaxChars = 220  // Max chars per memory
)
```
