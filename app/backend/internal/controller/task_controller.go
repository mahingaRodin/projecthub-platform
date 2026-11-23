package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/middleware"
	"qt-devops-go-mongo/backend/internal/service"
)

type TaskController struct {
	tasks service.TaskService
}

func NewTaskController(tasks service.TaskService) *TaskController {
	return &TaskController{tasks: tasks}
}

// List handles GET /api/projects/:id/tasks.
func (ctl *TaskController) List(c *gin.Context) {
	projectID, ok := pathID(c)
	if !ok {
		return
	}
	res, err := ctl.tasks.List(c.Request.Context(), middleware.CurrentUser(c), projectID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Create handles POST /api/projects/:id/tasks.
func (ctl *TaskController) Create(c *gin.Context) {
	projectID, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CreateTaskRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.tasks.Create(c.Request.Context(), middleware.CurrentUser(c), projectID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// Update handles PATCH /api/tasks/:id.
func (ctl *TaskController) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateTaskRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.tasks.Update(c.Request.Context(), middleware.CurrentUser(c), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Delete handles DELETE /api/tasks/:id.
func (ctl *TaskController) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := ctl.tasks.Delete(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
