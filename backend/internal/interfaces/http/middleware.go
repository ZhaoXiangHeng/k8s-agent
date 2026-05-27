package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"k8s-ai-ops/backend/internal/infra/auth"
)

// AuthMiddleware 适配 JWT 验证器为 Gin 中间件。
func AuthMiddleware(v *auth.JWTValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("requestId", requestID)
		user, err := v.ResolveForGin(c.Request, requestID)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"code": "UNAUTHENTICATED", "message": err.Error(), "requestId": requestID}})
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

// AdminRequired 校验当前用户是否为 admin。
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		if u, ok := user.(*auth.UserContext); !ok || u.Role != "admin" {
			c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Admin role required."}})
			return
		}
		c.Next()
	}
}

// RequestLogger 记录每个 HTTP 请求。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		pkgLog.WithFields(logrus.Fields{
			"event": "http_request_start", "method": c.Request.Method,
			"path": c.Request.URL.Path, "remote": c.ClientIP(),
		}).Info("HTTP request received")
		c.Next()
		pkgLog.WithFields(logrus.Fields{
			"event": "http_request_complete", "method": c.Request.Method,
			"path": c.Request.URL.Path, "status": c.Writer.Status(),
		}).Info("HTTP request completed")
	}
}

// getUser 提取当前认证用户信息。
func getUser(c *gin.Context) (userID, username, role string) {
	u, _ := c.Get("user")
	if uc, ok := u.(*auth.UserContext); ok {
		return uc.UserID, uc.Username, uc.Role
	}
	return "", "", ""
}
