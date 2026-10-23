package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/middleware"
	"qt-devops-go-mongo/backend/internal/service"
)

type AuthController struct {
	auth service.AuthService
}

func NewAuthController(auth service.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

// Register handles POST /api/auth/register.
func (ctl *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.auth.Register(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// Login handles POST /api/auth/login.
func (ctl *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.auth.Login(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Me handles GET /api/auth/me.
func (ctl *AuthController) Me(c *gin.Context) {
	c.JSON(http.StatusOK, dto.MeResponse{User: middleware.CurrentUser(c).Public()})
}
