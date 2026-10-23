package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/middleware"
	"qt-devops-go-mongo/backend/internal/service"
)

type UserController struct {
	users service.UserService
}

func NewUserController(users service.UserService) *UserController {
	return &UserController{users: users}
}

// List handles GET /api/users (admin, manager).
func (ctl *UserController) List(c *gin.Context) {
	users, err := ctl.users.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

// ChangeRole handles PATCH /api/users/:id/role (admin).
func (ctl *UserController) ChangeRole(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.RoleRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := ctl.users.ChangeRole(c.Request.Context(), middleware.CurrentUser(c), id, req.Role); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.RoleResponse{ID: id, Role: req.Role})
}

// Delete handles DELETE /api/users/:id (admin).
func (ctl *UserController) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := ctl.users.Delete(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
