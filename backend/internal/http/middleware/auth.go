package middleware

import (
	"cinema/internal/auth"
	"cinema/internal/model"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const CtxUserID = "user_id"
const CtxRole = "role"

func AuthRequired(jwtSvc *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":    false,
				"error": "invalid_token",
			})

			return
		}

		tokenStr := strings.TrimPrefix(h, "Bearer")
		claims, err := jwtSvc.Verify(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":    false,
				"error": "invalid_token",
			})

			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

func RequireRole(role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(CtxRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":    false,
				"error": "no_role",
			})

			return
		}

		if v.(model.UserRole) != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"ok":    false,
				"error": "forbidden",
			})

			return
		}

		c.Next()
	}
}
