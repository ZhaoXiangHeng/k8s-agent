package domain

import "time"

// ─── 值对象 ───

// MessageRole 是 Chat 消息的角色。
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

// ─── ChatSession 聚合根 ───

// ChatSession 是 Chat 会话聚合根，管理会话下的消息集合。
type ChatSession struct {
	ID        string
	UserID    string
	ModelID   string
	Title     string
	Status    string // "active" | "closed"
	Messages  []ChatMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewChatSession 创建新会话。
func NewChatSession(userID string) *ChatSession {
	return &ChatSession{
		UserID:   userID,
		Status:   "active",
		Messages: []ChatMessage{},
	}
}

// AddMessage 向会话追加一条消息。
func (s *ChatSession) AddMessage(role MessageRole, content string) ChatMessage {
	msg := ChatMessage{
		SessionID: s.ID,
		Role:      role,
		Content:   content,
	}
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
	return msg
}

// ─── ChatMessage 实体 ───

// ChatMessage 是 Chat 会话中的一条消息。
type ChatMessage struct {
	ID             string
	SessionID      string
	Role           MessageRole
	Content        string
	ToolName       string
	ToolArgsJSON   string
	ToolResultJSON string
	CreatedAt      time.Time
}

// IsToolCall 判断此消息是否为工具调用记录。
func (m ChatMessage) IsToolCall() bool { return m.ToolName != "" }
