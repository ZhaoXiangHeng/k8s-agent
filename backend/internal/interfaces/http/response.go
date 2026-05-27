package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ok(c *gin.Context, data any)      { c.JSON(http.StatusOK, data) }
func created(c *gin.Context, data any) { c.JSON(http.StatusCreated, data) }
func fail(c *gin.Context, s int, code, msg string) {
	requestID := requestIDFromContext(c)
	c.AbortWithStatusJSON(s, gin.H{"error": gin.H{"code": code, "message": msg, "requestId": requestID}})
}
func failBadRequest(c *gin.Context, code, msg string) { fail(c, 400, code, msg) }
func failForbidden(c *gin.Context, code, msg string)  { fail(c, 403, code, msg) }
func failNotFound(c *gin.Context, code, msg string)   { fail(c, 404, code, msg) }
func failInternal(c *gin.Context, code, msg string)   { fail(c, 500, code, msg) }
func failBadGateway(c *gin.Context, code, msg string) { fail(c, 502, code, msg) }

func requestIDFromContext(c *gin.Context) string {
	if value, ok := c.Get("requestId"); ok {
		if requestID, ok := value.(string); ok && requestID != "" {
			return requestID
		}
	}
	if requestID := c.Writer.Header().Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	return uuid.New().String()
}
