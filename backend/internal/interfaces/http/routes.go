package http

import (
	"github.com/gin-gonic/gin"

	"k8s-ai-ops/backend/internal/infra/auth"
)

func RegisterRoutes(r *gin.Engine, srv *Server, jwtValidator *auth.JWTValidator) {
	r.Use(RequestLogger())
	r.Use(AuthMiddleware(jwtValidator))

	r.GET("/healthz", healthz)

	op := r.Group("/api")
	{
		op.GET("/me", srv.currentUser)
		op.GET("/operator/permissions", srv.operatorPermissions)
		op.GET("/operator/llm-models", srv.operatorModels)
		op.POST("/operator/chat/sessions", srv.createChatSession)
		op.POST("/operator/chat/sessions/:sessionId/messages", srv.createChatMessage)
	}

	admin := r.Group("/api/admin", AdminRequired())
	{
		admin.GET("/users", srv.listUsers)
		admin.POST("/users", srv.createUser)
		admin.PUT("/users/:userId/permissions", srv.updatePermissions)
		admin.GET("/audit-logs", srv.listAuditLogs)
		admin.GET("/llm/providers", srv.listProviders)
		admin.POST("/llm/providers", srv.createProvider)
		admin.PUT("/llm/providers/:id", srv.updateProvider)
		admin.PATCH("/llm/providers/:id", srv.updateProvider)
		admin.GET("/llm/models", srv.listModels)
		admin.POST("/llm/models", srv.createModel)
		admin.PUT("/llm/models/:id", srv.updateModel)
		admin.PATCH("/llm/models/:id", srv.updateModel)
	}
}
