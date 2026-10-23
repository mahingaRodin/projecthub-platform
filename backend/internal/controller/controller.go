// Package controller contains the HTTP layer: it binds requests, calls services and writes responses.
// Controllers hold no business rules; those live in the service package.
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/apperror"
)

// respondError writes {"error": "..."} with the status carried by an apperror; anything else is a 500.
func respondError(c *gin.Context, err error) {
	status, msg := apperror.HTTP(err)
	if status >= http.StatusInternalServerError {
		_ = c.Error(err)
	}
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

func bindJSON(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func pathID(c *gin.Context) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return id, false
	}
	return id, true
}
