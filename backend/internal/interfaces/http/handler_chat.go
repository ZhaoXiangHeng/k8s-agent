package http

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/app"
)

type sseWriter struct {
	w       gin.ResponseWriter
	flusher http.Flusher
}

func (s *sseWriter) WriteSSE(data []byte) {
	s.w.Write([]byte("data: "))
	s.w.Write(data)
	s.w.Write([]byte("\n\n"))
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func (s *Server) createChatSession(c *gin.Context) {
	uid, _, _ := getUser(c)
	result, err := s.Svc.Chat.CreateSession(c.Request.Context(), uid)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	created(c, result)
}

func (s *Server) createChatMessage(c *gin.Context) {
	var req app.ChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		failBadRequest(c, "INVALID_REQUEST", "Invalid request body.")
		return
	}

	uid, username, role := getUser(c)
	sessionID := c.Param("sessionId")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, _ := c.Writer.(http.Flusher)
	sender := &sseWriter{w: c.Writer, flusher: flusher}

	if err := s.Svc.Chat.ProcessMessage(c.Request.Context(), req, uid, username, role, sessionID, sender); err != nil {
		if auditErr := s.Svc.Audit.Record(c.Request.Context(), uid, "operator.chat.message", "chat_message", "", false, err.Error()); auditErr != nil {
			pkgLog.WithError(auditErr).WithField("event", "audit_record_failed").Error("failed to record chat audit log")
		}
		payload, _ := json.Marshal(gin.H{"error": err.Error()})
		c.Writer.Write([]byte("data: "))
		c.Writer.Write(payload)
		c.Writer.Write([]byte("\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
		return
	}
	if auditErr := s.Svc.Audit.Record(c.Request.Context(), uid, "operator.chat.message", "chat_message", "", true, "completed"); auditErr != nil {
		pkgLog.WithError(auditErr).WithField("event", "audit_record_failed").Error("failed to record chat audit log")
	}
}
