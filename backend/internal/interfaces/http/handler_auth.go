package http

import (
	"k8s-ai-ops/backend/internal/app"

	"github.com/gin-gonic/gin"
)

func (s *Server) currentUser(c *gin.Context) {
	uid, username, role := getUser(c)
	c.JSON(200, s.Svc.Users.GetCurrentInfo(c.Request.Context(), uid, username, role))
}

func (s *Server) operatorPermissions(c *gin.Context) {
	uid, _, role := getUser(c)
	// admin 拥有集群管理员权限，返回通配符
	if role == "admin" {
		ok(c, []app.PermissionResponse{{
			Namespace: "*", APIGroup: "*", Resource: "*",
			Verbs: []string{"*"}, Enabled: true,
		}})
		return
	}
	result, err := s.Svc.Users.GetPermissions(c.Request.Context(), uid)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}

func (s *Server) operatorModels(c *gin.Context) {
	uid, _, role := getUser(c)
	// admin 不受模型绑定限制，可查看所有已启用的模型
	if role == "admin" {
		uid = ""
	}
	result, err := s.Svc.LLM.ListModels(c.Request.Context(), uid, true)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}
