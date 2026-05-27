package http

import (
	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
)

func (s *Server) listProviders(c *gin.Context) {
	result, err := s.Svc.LLM.ListProviders(c.Request.Context())
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}

func (s *Server) createProvider(c *gin.Context) {
	var req app.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	result, err := s.Svc.LLM.CreateProvider(c.Request.Context(), req)
	if err != nil {
		failBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.provider.create", "llm_provider", result.ID, true, "created") {
		return
	}
	created(c, result)
}

func (s *Server) updateProvider(c *gin.Context) {
	var req app.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	result, err := s.Svc.LLM.UpdateProvider(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		failNotFound(c, "NOT_FOUND", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.provider.update", "llm_provider", result.ID, true, "updated") {
		return
	}
	ok(c, result)
}
