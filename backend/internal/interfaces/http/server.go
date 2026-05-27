package http

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/app"
)

var pkgLog = logrus.WithField("component", "backend-api/http")

// Server 是 HTTP API 的核心依赖容器，持有应用服务层。
type Server struct {
	Svc *app.Services
}

// NewServer 创建 Server，svc 是必需依赖。
func NewServer(svc *app.Services) *Server {
	return &Server{Svc: svc}
}

func (s *Server) recordAuditOrFail(c *gin.Context, actorID, action, targetType, targetID string, allowed bool, reason string) bool {
	if err := s.Svc.Audit.Record(c.Request.Context(), actorID, action, targetType, targetID, allowed, reason); err != nil {
		pkgLog.WithError(err).WithFields(logrus.Fields{
			"event":       "audit_record_failed",
			"actor_id":    actorID,
			"action":      action,
			"target_type": targetType,
			"target_id":   targetID,
		}).Error("failed to record audit log")
		failInternal(c, "AUDIT_RECORD_FAILED", "Failed to record audit log.")
		return false
	}
	return true
}
