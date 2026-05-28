package http

import (
	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
)

func (s *Server) listModels(c *gin.Context) {
	result, err := s.Svc.LLM.ListModels(c.Request.Context(), "", false)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}

func (s *Server) createModel(c *gin.Context) {
	var req app.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	actorID, _, _ := getUser(c)
	result, err := s.Svc.LLM.CreateModel(c.Request.Context(), req, actorID)
	if err != nil {
		failBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}
	if !s.recordAuditOrFail(c, actorID, "admin.model.create", "llm_model", result.ID, true, "created") {
		return
	}
	created(c, result)
}

func (s *Server) updateModel(c *gin.Context) {
	var req app.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}
	result, err := s.Svc.LLM.UpdateModel(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		failNotFound(c, "NOT_FOUND", err.Error())
		return
	}
	actorID, _, _ := getUser(c)
	if !s.recordAuditOrFail(c, actorID, "admin.model.update", "llm_model", result.ID, true, "updated") {
		return
	}
	ok(c, result)
}
