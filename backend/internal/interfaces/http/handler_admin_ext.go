package http

import (
	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
)

// ─── User management extensions ───

func (s *Server) deleteUser(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		failBadRequest(c, "INVALID_REQUEST", "userId is required")
		return
	}
	if err := s.Svc.Users.Delete(c.Request.Context(), userID); err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.user.delete", "user", userID, true, "deleted") {
		return
	}
	ok(c, gin.H{"status": "deleted"})
}

func (s *Server) resetUserPassword(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		failBadRequest(c, "INVALID_REQUEST", "userId is required")
		return
	}
	var req app.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	if err := s.Svc.Users.ResetPassword(c.Request.Context(), userID, req.Password); err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.user.reset_password", "user", userID, true, "password reset") {
		return
	}
	ok(c, gin.H{"status": "ok"})
}

// ─── Model management extensions ───

func (s *Server) deleteModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		failBadRequest(c, "INVALID_REQUEST", "model id is required")
		return
	}
	if err := s.Svc.LLM.DeleteModel(c.Request.Context(), modelID); err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.llm.delete_model", "model", modelID, true, "deleted") {
		return
	}
	ok(c, gin.H{"status": "deleted"})
}

// ─── Model bindings ───

func (s *Server) listModelBindings(c *gin.Context) {
	result, err := s.Svc.Users.GetAllModelBindings(c.Request.Context())
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}

func (s *Server) updateModelBindings(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		failBadRequest(c, "INVALID_REQUEST", "userId is required")
		return
	}
	var req app.UpdateModelBindingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	if err := s.Svc.Users.UpdateModelBindings(c.Request.Context(), userID, req.ModelIDs); err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.llm.update_bindings", "user", userID, true, "model bindings updated") {
		return
	}
	ok(c, gin.H{"status": "ok"})
}

// ─── Chat session management ───

func (s *Server) deleteChatSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		failBadRequest(c, "INVALID_REQUEST", "sessionId is required")
		return
	}
	uid, _, _ := getUser(c)
	if err := s.Svc.Chat.DeleteSession(c.Request.Context(), sessionID, uid); err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, gin.H{"status": "deleted"})
}
