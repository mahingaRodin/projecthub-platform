package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/apperror"
	"qt-devops-go-mongo/backend/internal/cache"
	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/repository"
)

const dateLayout = "2006-01-02"

type TaskService interface {
	List(ctx context.Context, actor *models.User, projectID primitive.ObjectID) ([]models.Task, error)
	// Create and Delete are allowed for the project owner and admins.
	Create(ctx context.Context, actor *models.User, projectID primitive.ObjectID, req dto.CreateTaskRequest) (*models.Task, error)
	// Update: owners and admins may change any field; the assignee may only change the status.
	Update(ctx context.Context, actor *models.User, taskID primitive.ObjectID, req dto.UpdateTaskRequest) (*models.Task, error)
	Delete(ctx context.Context, actor *models.User, taskID primitive.ObjectID) error
}

type taskService struct {
	tasks    *repository.TaskRepository
	projects *repository.ProjectRepository
	cache    cache.Cache
}

func NewTaskService(tasks *repository.TaskRepository, projects *repository.ProjectRepository, c cache.Cache) TaskService {
	return &taskService{tasks: tasks, projects: projects, cache: c}
}

func (s *taskService) List(ctx context.Context, actor *models.User, projectID primitive.ObjectID) ([]models.Task, error) {
	p, err := loadVisibleProject(ctx, s.projects, actor, projectID)
	if err != nil {
		return nil, err
	}
	return s.tasks.FindByProject(ctx, p.ID)
}

func (s *taskService) Create(ctx context.Context, actor *models.User, projectID primitive.ObjectID, req dto.CreateTaskRequest) (*models.Task, error) {
	p, err := loadVisibleProject(ctx, s.projects, actor, projectID)
	if err != nil {
		return nil, err
	}
	if !canManage(actor, p) {
		return nil, apperror.Forbidden("only the project owner or an admin can create tasks")
	}
	if req.AssigneeID != nil && !isParticipant(p, *req.AssigneeID) {
		return nil, apperror.BadRequest("assignee must be the project owner or a project member")
	}

	now := time.Now().UTC()
	t := &models.Task{
		ID:          primitive.NewObjectID(),
		ProjectID:   p.ID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      orDefault(req.Status, "todo"),
		Priority:    orDefault(req.Priority, "medium"),
		AssigneeID:  req.AssigneeID,
		CreatedBy:   actor.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.DueDate != "" {
		d, _ := time.Parse(dateLayout, req.DueDate) // format already validated by the request binding
		t.DueDate = &d
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, s.touch(ctx, p.ID)
}

func (s *taskService) Update(ctx context.Context, actor *models.User, taskID primitive.ObjectID, req dto.UpdateTaskRequest) (*models.Task, error) {
	t, p, err := s.load(ctx, actor, taskID)
	if err != nil {
		return nil, err
	}
	if !canManage(actor, p) {
		if t.AssigneeID == nil || *t.AssigneeID != actor.ID {
			return nil, apperror.Forbidden("you can only update tasks assigned to you")
		}
		if !req.OnlyStatus() {
			return nil, apperror.Forbidden("assignees can only change the task status")
		}
	}

	if req.Title != nil {
		t.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		t.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID == "" {
			t.AssigneeID = nil
		} else {
			aid, err := primitive.ObjectIDFromHex(*req.AssigneeID)
			if err != nil || !isParticipant(p, aid) {
				return nil, apperror.BadRequest("assignee must be the project owner or a project member")
			}
			t.AssigneeID = &aid
		}
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			t.DueDate = nil
		} else {
			d, err := time.Parse(dateLayout, *req.DueDate)
			if err != nil {
				return nil, apperror.BadRequest("dueDate must be in YYYY-MM-DD format")
			}
			t.DueDate = &d
		}
	}
	t.UpdatedAt = time.Now().UTC()

	if err := s.tasks.Replace(ctx, t); err != nil {
		return nil, err
	}
	return t, s.touch(ctx, p.ID)
}

func (s *taskService) Delete(ctx context.Context, actor *models.User, taskID primitive.ObjectID) error {
	t, p, err := s.load(ctx, actor, taskID)
	if err != nil {
		return err
	}
	if !canManage(actor, p) {
		return apperror.Forbidden("only the project owner or an admin can delete tasks")
	}
	if err := s.tasks.Delete(ctx, t.ID); err != nil {
		return err
	}
	return s.touch(ctx, p.ID)
}

// touch records that a project's tasks changed: bumps updatedAt and invalidates cached dashboards.
func (s *taskService) touch(ctx context.Context, projectID primitive.ObjectID) error {
	s.cache.Bump(ctx, cache.StatsGroup)
	return s.projects.Touch(ctx, projectID)
}

// load returns a task and its project, if the actor may see the project.
func (s *taskService) load(ctx context.Context, actor *models.User, taskID primitive.ObjectID) (*models.Task, *models.Project, error) {
	t, err := s.tasks.FindByID(ctx, taskID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, apperror.NotFound("task not found")
	}
	if err != nil {
		return nil, nil, err
	}
	p, err := loadVisibleProject(ctx, s.projects, actor, t.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	return t, p, nil
}
