package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/middleware"
	"qt-devops-go-mongo/backend/internal/service"
)

type ProjectController struct {
	projects service.ProjectService
}

func NewProjectController(projects service.ProjectService) *ProjectController {
	return &ProjectController{projects: projects}
}

// List handles GET /api/projects.
func (ctl *ProjectController) List(c *gin.Context) {
	res, err := ctl.projects.List(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Get handles GET /api/projects/:id.
func (ctl *ProjectController) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	res, err := ctl.projects.Get(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Create handles POST /api/projects (admin, manager).
func (ctl *ProjectController) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.projects.Create(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// Update handles PUT /api/projects/:id.
func (ctl *ProjectController) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateProjectRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.projects.Update(c.Request.Context(), middleware.CurrentUser(c), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Delete handles DELETE /api/projects/:id.
func (ctl *ProjectController) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := ctl.projects.Delete(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
