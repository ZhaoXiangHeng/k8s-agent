package domain

import (
	"fmt"
	"strings"
	"time"
)

// ─── 值对象 ───

// Protocol 是 LLM API 协议类型。
type Protocol string

const (
	ProtocolOpenAI    Protocol = "openai"
	ProtocolAnthropic Protocol = "anthropic"
)

// NewProtocol 创建协议值对象。
func NewProtocol(s string) (Protocol, error) {
	p := Protocol(strings.ToLower(strings.TrimSpace(s)))
	switch p {
	case ProtocolOpenAI, ProtocolAnthropic:
		return p, nil
	}
	return "", fmt.Errorf("%w: invalid protocol %q", ErrInvalidInput, s)
}

// ─── 实体 ───

// LLMProvider 是 LLM API Provider 的配置。
type LLMProvider struct {
	ID               string
	Name             string
	Protocol         Protocol
	BaseURL          string
	APIKeyCiphertext string // AES-256-GCM 加密后的密文
	Enabled          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Disable 停用 Provider。
func (p *LLMProvider) Disable() { p.Enabled = false; p.UpdatedAt = time.Now() }

// Enable 启用 Provider。
func (p *LLMProvider) Enable() { p.Enabled = true; p.UpdatedAt = time.Now() }

// LLMModel 是 LLM Model 的配置。
type LLMModel struct {
	ID                string
	ProviderID        string
	ModelName         string
	DisplayName       string
	SupportsTools     bool
	SupportsStreaming bool
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Disable 停用 Model。
func (m *LLMModel) Disable() { m.Enabled = false; m.UpdatedAt = time.Now() }

// Enable 启用 Model。
func (m *LLMModel) Enable() { m.Enabled = true; m.UpdatedAt = time.Now() }
