// Package router maps URLs to controllers and applies middleware (CORS, authentication, role guards).
package router

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/config"
	"qt-devops-go-mongo/backend/internal/controller"
	"qt-devops-go-mongo/backend/internal/middleware"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/service"
)

type Controllers struct {
	Auth     *controller.AuthController
	Users    *controller.UserController
	Projects *controller.ProjectController
	Tasks    *controller.TaskController
	Stats    *controller.StatsController
	Health   *controller.HealthController
}

func New(cfg config.Config, auth service.AuthService, ctl Controllers) *gin.Engine {
	// Gin reads GIN_MODE before .env is loaded, so apply it again here.
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSOrigins))
	_ = r.SetTrustedProxies(nil)

	admin, manager := models.RoleAdmin, models.RoleManager

	api := r.Group("/api")
	api.GET("/health", ctl.Health.Check)
	api.POST("/auth/register", ctl.Auth.Register)
	api.POST("/auth/login", ctl.Auth.Login)

	authed := api.Group("", middleware.Authenticate(auth))
	authed.GET("/auth/me", ctl.Auth.Me)
	authed.GET("/stats", ctl.Stats.Get)

	authed.GET("/users", middleware.RequireRoles(admin, manager), ctl.Users.List)
	authed.PATCH("/users/:id/role", middleware.RequireRoles(admin), ctl.Users.ChangeRole)
	authed.DELETE("/users/:id", middleware.RequireRoles(admin), ctl.Users.Delete)

	authed.GET("/projects", ctl.Projects.List)
	authed.POST("/projects", middleware.RequireRoles(admin, manager), ctl.Projects.Create)
	authed.GET("/projects/:id", ctl.Projects.Get)
	authed.PUT("/projects/:id", ctl.Projects.Update)
	authed.DELETE("/projects/:id", ctl.Projects.Delete)

	authed.GET("/projects/:id/tasks", ctl.Tasks.List)
	authed.POST("/projects/:id/tasks", ctl.Tasks.Create)
	authed.PATCH("/tasks/:id", ctl.Tasks.Update)
	authed.DELETE("/tasks/:id", ctl.Tasks.Delete)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})
	return r
}
