// Package middleware contains Gin middleware for authentication and role-based authorization.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/apperror"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/service"
)

const currentUserKey = "currentUser"

// Authenticate requires "Authorization: Bearer <jwt>" and stores the authenticated user in the context.
func Authenticate(auth service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		user, err := auth.Authenticate(c.Request.Context(), token)
		if err != nil {
			status, msg := apperror.HTTP(err)
			if status >= http.StatusInternalServerError {
				_ = c.Error(err)
			}
			c.AbortWithStatusJSON(status, gin.H{"error": msg})
			return
		}
		c.Set(currentUserKey, user)
		c.Next()
	}
}

// RequireRoles must run after Authenticate.
func RequireRoles(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		for _, r := range roles {
			if u.Role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "your role is not allowed to perform this action"})
	}
}

// CurrentUser returns the user set by Authenticate.
func CurrentUser(c *gin.Context) *models.User {
	return c.MustGet(currentUserKey).(*models.User)
}
