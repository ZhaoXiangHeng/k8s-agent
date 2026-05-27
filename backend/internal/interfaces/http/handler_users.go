package http

import (
	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
)

func (s *Server) listUsers(c *gin.Context) {
	result, err := s.Svc.Users.List(c.Request.Context())
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}

func (s *Server) createUser(c *gin.Context) {
	var req app.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	result, err := s.Svc.Users.Create(c.Request.Context(), req)
	if err != nil {
		failBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.user.create", "user", result.ID, true, "created") {
		return
	}
	created(c, result)
}
