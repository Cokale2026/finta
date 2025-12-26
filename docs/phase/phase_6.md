## Phase 6: Session 管理 (2 天)

### 目标

实现会话持久化和上下文管理，支持长时间对话。

### 实现步骤

#### 6.1 Session 接口

**文件**: `pkg/session/session.go`

```go
package session

import (
    "context"
    "finta/internal/llm"
    "time"
)

type Session interface {
    ID() string
    AddMessage(msg llm.Message) error
    GetMessages() []llm.Message
    Save(ctx context.Context) error
    Load(ctx context.Context, sessionID string) error
}

type SessionData struct {
    ID        string
    Messages  []llm.Message
    Metadata  map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

#### 6.2 SQLite 持久化

**文件**: `pkg/session/persistence.go`

使用 SQLite 存储会话数据：

```go
package session

import (
    "context"
    "database/sql"
    "encoding/json"

    _ "github.com/mattn/go-sqlite3"
)

type SQLitePersistence struct {
    db *sql.DB
}

func NewSQLitePersistence(dbPath string) (*SQLitePersistence, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // 创建表
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,
            messages TEXT,
            metadata TEXT,
            created_at DATETIME,
            updated_at DATETIME
        )
    `)
    if err != nil {
        return nil, err
    }

    return &SQLitePersistence{db: db}, nil
}

func (p *SQLitePersistence) Save(ctx context.Context, data *SessionData) error {
    messagesJSON, _ := json.Marshal(data.Messages)
    metadataJSON, _ := json.Marshal(data.Metadata)

    _, err := p.db.ExecContext(ctx, `
        INSERT OR REPLACE INTO sessions (id, messages, metadata, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?)
    `, data.ID, messagesJSON, metadataJSON, data.CreatedAt, data.UpdatedAt)

    return err
}

func (p *SQLitePersistence) Load(ctx context.Context, sessionID string) (*SessionData, error) {
    // 实现加载逻辑
}
```

#### 6.3 Context Summarization

**文件**: `pkg/session/summarizer.go`

当消息过多时，使用 LLM 生成摘要：

```go
package session

import (
    "context"
    "finta/internal/llm"
)

type Summarizer struct {
    llmClient llm.Client
}

func (s *Summarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
    // 使用 LLM 生成对话摘要
}
```

### Phase 6 完成标准

- ✅ Session 接口和基础实现
- ✅ SQLite 持久化
- ✅ 会话可以保存和加载
- ✅ 上下文摘要功能
- ✅ CLI 支持恢复历史会话

---
