package http

import "github.com/gin-gonic/gin"

func (s *Server) listAuditLogs(c *gin.Context) {
	result, err := s.Svc.Audit.List(c.Request.Context())
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}
