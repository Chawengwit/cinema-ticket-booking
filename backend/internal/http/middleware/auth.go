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
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":    false,
				"error": "missing_authorization_header",
			})
			return
		}

		// Expect: "Bearer <token>"
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":    false,
				"error": "invalid_authorization_format",
			})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])

		claims, err := jwtSvc.Verify(tokenStr)
		if err != nil {
			// debug-friendly (เอา detail ออกได้ตอน production)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":     false,
				"error":  "invalid_token",
				"detail": err.Error(),
			})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		// store role as string for easy JSON + GetString
		c.Set(CtxRole, string(claims.Role))

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

		roleStr, ok := v.(string)
		if !ok || roleStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":    false,
				"error": "invalid_role_context",
			})
			return
		}

		if model.UserRole(roleStr) != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		c.Next()
	}
}
