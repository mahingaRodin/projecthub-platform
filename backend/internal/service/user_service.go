package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/apperror"
	"qt-devops-go-mongo/backend/internal/cache"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/repository"
)

type UserService interface {
	List(ctx context.Context) ([]models.PublicUser, error)
	ChangeRole(ctx context.Context, actor *models.User, id primitive.ObjectID, role models.Role) error
	// Delete removes a user, transfers the projects they own to the acting admin and unassigns their tasks.
	Delete(ctx context.Context, actor *models.User, id primitive.ObjectID) error
}

type userService struct {
	users    *repository.UserRepository
	projects *repository.ProjectRepository
	tasks    *repository.TaskRepository
	cache    cache.Cache
}

func NewUserService(users *repository.UserRepository, projects *repository.ProjectRepository, tasks *repository.TaskRepository, c cache.Cache) UserService {
	return &userService{users: users, projects: projects, tasks: tasks, cache: c}
}

func (s *userService) List(ctx context.Context) ([]models.PublicUser, error) {
	users, err := s.users.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.PublicUser, 0, len(users))
	for _, u := range users {
		out = append(out, u.Public())
	}
	return out, nil
}

func (s *userService) ChangeRole(ctx context.Context, actor *models.User, id primitive.ObjectID, role models.Role) error {
	if id == actor.ID {
		return apperror.BadRequest("you cannot change your own role")
	}
	err := s.users.UpdateRole(ctx, id, role)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.NotFound("user not found")
	}
	if err != nil {
		return err
	}
	// The new role must apply on the user's next request, and admins see different dashboards.
	s.cache.Delete(ctx, cache.UserKey(id))
	s.cache.Bump(ctx, cache.StatsGroup)
	return nil
}

func (s *userService) Delete(ctx context.Context, actor *models.User, id primitive.ObjectID) error {
	if id == actor.ID {
		return apperror.BadRequest("you cannot delete your own account")
	}
	if err := s.users.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.NotFound("user not found")
		}
		return err
	}
	if err := s.projects.TransferOwnership(ctx, id, actor.ID); err != nil {
		return err
	}
	if err := s.projects.RemoveMember(ctx, id); err != nil {
		return err
	}
	if err := s.tasks.Unassign(ctx, id); err != nil {
		return err
	}
	s.cache.Delete(ctx, cache.UserKey(id))
	s.cache.Bump(ctx, cache.StatsGroup)
	return nil
}
