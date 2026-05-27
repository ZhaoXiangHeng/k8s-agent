package http

import "github.com/gin-gonic/gin"

func (s *Server) currentUser(c *gin.Context) {
	uid, username, role := getUser(c)
	c.JSON(200, s.Svc.Users.GetCurrentInfo(c.Request.Context(), uid, username, role))
}

func (s *Server) operatorPermissions(c *gin.Context) {
	uid, _, _ := getUser(c)
	result, err := s.Svc.Users.GetPermissions(c.Request.Context(), uid)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}

func (s *Server) operatorModels(c *gin.Context) {
	uid, _, _ := getUser(c)
	result, err := s.Svc.LLM.ListModels(c.Request.Context(), uid, true)
	if err != nil {
		failInternal(c, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, result)
}
