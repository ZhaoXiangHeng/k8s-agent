// Package app 是 Backend 的应用服务层，按业务域拆分为独立文件。
package app

import (
	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/domain"
)

// appLog 是应用服务层的统一日志记录器。
var appLog = logrus.WithField("layer", "app")

// Services 聚合全部应用服务。
type Services struct {
	Users       *UserService
	Permissions *PermissionService
	LLM         *LLMService
	Chat        *ChatService
	Audit       *AuditService
}

// NewServices 创建应用服务聚合。ChatService 因需要 Agent 端口，通过 SetChatService 单独注入。
func NewServices(repos *domain.Repositories, cipher SecretCipher, rbacApplier RBACApplier) *Services {
	return &Services{
		Users:       &UserService{repos: repos},
		Permissions: NewPermissionService(repos, rbacApplier),
		LLM:         NewLLMService(repos, cipher),
		Audit:       &AuditService{repos: repos},
	}
}

// SetChatService 注入 Chat 应用服务。
func (s *Services) SetChatService(cs *ChatService) { s.Chat = cs }
