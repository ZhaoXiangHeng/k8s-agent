package http

import (
	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
)

func (s *Server) updatePermissions(c *gin.Context) {
	userID := c.Param("userId")
	var req app.UpdatePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	result, err := s.Svc.Permissions.Update(c.Request.Context(), userID, req)
	if err != nil {
		failInternal(c, "K8S_RBAC_APPLY_FAILED", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.permissions.update", "user", userID, true, "updated") {
		return
	}
	ok(c, result)
}

func (s *Server) getUserPermissions(c *gin.Context) {
	userID := c.Param("userId")
	result, err := s.Svc.Users.GetPermissions(c.Request.Context(), userID)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}
