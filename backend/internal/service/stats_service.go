package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/cache"
	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/repository"
)

type StatsService interface {
	// Get returns dashboard figures scoped to the projects the actor can see.
	Get(ctx context.Context, actor *models.User) (*dto.StatsResponse, error)
}

type statsService struct {
	projects *repository.ProjectRepository
	tasks    *repository.TaskRepository
	cache    cache.Cache
	ttl      time.Duration
}

func NewStatsService(projects *repository.ProjectRepository, tasks *repository.TaskRepository, c cache.Cache, ttl time.Duration) StatsService {
	return &statsService{projects: projects, tasks: tasks, cache: c, ttl: ttl}
}

// Get serves the dashboard from the cache when possible. Keys include the "stats" version, which every
// project/task/user change bumps, so one write invalidates all users' cached dashboards at once.
func (s *statsService) Get(ctx context.Context, actor *models.User) (*dto.StatsResponse, error) {
	version := s.cache.Version(ctx, cache.StatsGroup)
	if version >= 0 {
		var cached dto.StatsResponse
		if s.cache.Get(ctx, cache.StatsKey(version, actor.ID), &cached) {
			return &cached, nil
		}
	}
	res, err := s.compute(ctx, actor)
	if err != nil {
		return nil, err
	}
	if version >= 0 {
		s.cache.Set(ctx, cache.StatsKey(version, actor.ID), res, s.ttl)
	}
	return res, nil
}

func (s *statsService) compute(ctx context.Context, actor *models.User) (*dto.StatsResponse, error) {
	var (
		projects []models.Project
		err      error
	)
	if actor.Role == models.RoleAdmin {
		projects, err = s.projects.FindAll(ctx)
	} else {
		projects, err = s.projects.FindByParticipant(ctx, actor.ID)
	}
	if err != nil {
		return nil, err
	}

	ids := []primitive.ObjectID{}
	names := map[primitive.ObjectID]string{}
	byStatus := map[string]int{"planned": 0, "active": 0, "completed": 0}
	for _, p := range projects {
		ids = append(ids, p.ID)
		names[p.ID] = p.Name
		byStatus[p.Status]++
	}

	tasksByStatus := map[string]int{"todo": 0, "in_progress": 0, "done": 0}
	counts, err := s.tasks.CountByStatus(ctx, ids)
	if err != nil {
		return nil, err
	}
	for status, n := range counts {
		tasksByStatus[status] = n
	}

	overdue, err := s.tasks.CountOverdue(ctx, ids, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	assigned, err := s.tasks.FindOpenAssignedTo(ctx, actor.ID, 10)
	if err != nil {
		return nil, err
	}
	mine := make([]dto.MyTask, 0, len(assigned))
	for _, t := range assigned {
		mine = append(mine, dto.MyTask{Task: t, ProjectName: names[t.ProjectID]})
	}

	return &dto.StatsResponse{
		Projects:         len(projects),
		ProjectsByStatus: byStatus,
		TasksByStatus:    tasksByStatus,
		OverdueTasks:     overdue,
		MyOpenTasks:      mine,
	}, nil
}
