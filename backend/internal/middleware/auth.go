package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/response"
	"personal_assistant_server/internal/service"
)

const ClaimsKey = "authClaims"

func Auth(auth *service.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, http.StatusUnauthorized, "请先登录")
			return
		}
		claims, err := auth.Parse(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "登录状态已失效")
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func CurrentClaims(c *gin.Context) (*service.Claims, bool) {
	value, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*service.Claims)
	return claims, ok
}
