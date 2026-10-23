package service

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"qt-devops-go-mongo/backend/internal/apperror"
	"qt-devops-go-mongo/backend/internal/cache"
	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/repository"
)

// Demo accounts, documented in the README ("Test accounts").
var seedUsers = []struct {
	dto.SeedUser
	password string
}{
	{dto.SeedUser{Name: "Alice Admin", Email: "admin@qt.rw", Role: models.RoleAdmin}, "Admin@123"},
	{dto.SeedUser{Name: "Mark Manager", Email: "manager@qt.rw", Role: models.RoleManager}, "Manager@123"},
	{dto.SeedUser{Name: "Mary Member", Email: "member@qt.rw", Role: models.RoleMember}, "Member@123"},
}

// The sample project has a fixed id so that API replicas seeding at the same time cannot create it twice.
var seedProjectID, _ = primitive.ObjectIDFromHex("5eed00000000000000000001")

type SeedService interface {
	// Seed is idempotent: it creates the demo users and a sample project only if they are missing.
	// It is called from main.go on the first start; it is not exposed over HTTP.
	Seed(ctx context.Context) (*dto.SeedResponse, error)
}

type seedService struct {
	enabled  bool
	users    *repository.UserRepository
	projects *repository.ProjectRepository
	tasks    *repository.TaskRepository
	cache    cache.Cache
}

func NewSeedService(enabled bool, users *repository.UserRepository, projects *repository.ProjectRepository, tasks *repository.TaskRepository, c cache.Cache) SeedService {
	return &seedService{enabled: enabled, users: users, projects: projects, tasks: tasks, cache: c}
}

func (s *seedService) Seed(ctx context.Context) (*dto.SeedResponse, error) {
	if !s.enabled {
		return nil, apperror.Forbidden("seeding is disabled (SEED_ENABLED=false)")
	}

	ids := map[models.Role]primitive.ObjectID{}
	created := 0
	accounts := make([]dto.SeedUser, 0, len(seedUsers))
	for _, su := range seedUsers {
		accounts = append(accounts, su.SeedUser)
		existing, err := s.users.FindByEmail(ctx, su.Email)
		if err == nil {
			ids[su.Role] = existing.ID
			continue
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(su.password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u := &models.User{
			ID: primitive.NewObjectID(), Name: su.Name, Email: su.Email,
			PasswordHash: string(hash), Role: su.Role, CreatedAt: time.Now().UTC(),
		}
		if err := s.users.Create(ctx, u); err != nil {
			if !errors.Is(err, repository.ErrDuplicate) {
				return nil, err
			}
			// Another API instance created this user a moment ago.
			existing, err := s.users.FindByEmail(ctx, su.Email)
			if err != nil {
				return nil, err
			}
			ids[su.Role] = existing.ID
			continue
		}
		ids[su.Role] = u.ID
		created++
	}

	n, err := s.projects.Count(ctx)
	if err != nil {
		return nil, err
	}
	projectCreated := false
	if n == 0 {
		projectCreated, err = s.seedProject(ctx, ids[models.RoleManager], ids[models.RoleMember])
		if err != nil {
			return nil, err
		}
	}

	if created > 0 || projectCreated {
		s.cache.Bump(ctx, cache.StatsGroup)
	}
	return &dto.SeedResponse{
		Message:        "seed completed",
		UsersCreated:   created,
		ProjectCreated: projectCreated,
		Users:          accounts,
	}, nil
}

// seedProject reports false if another API instance created the sample project first.
func (s *seedService) seedProject(ctx context.Context, manager, member primitive.ObjectID) (bool, error) {
	now := time.Now().UTC()
	p := &models.Project{
		ID:          seedProjectID,
		Name:        "QT Platform Migration",
		Description: "Move the QT platform from VMs to Kubernetes.",
		Status:      "active",
		OwnerID:     manager,
		MemberIDs:   []primitive.ObjectID{member},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.projects.Create(ctx, p); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return false, nil
		}
		return false, err
	}

	due := func(days int) *time.Time { d := now.AddDate(0, 0, days).Truncate(24 * time.Hour); return &d }
	tasks := []models.Task{
		{Title: "Provision VMs with Terraform", Status: "done", Priority: "high", AssigneeID: &manager, DueDate: due(-3)},
		{Title: "Containerise the API", Status: "in_progress", Priority: "high", AssigneeID: &member, DueDate: due(2)},
		{Title: "Write Kubernetes manifests", Status: "todo", Priority: "medium", AssigneeID: &member, DueDate: due(5)},
		{Title: "Set up monitoring", Status: "todo", Priority: "low"},
	}
	for i := range tasks {
		tasks[i].ID, tasks[i].ProjectID, tasks[i].CreatedBy = primitive.NewObjectID(), p.ID, manager
		tasks[i].CreatedAt, tasks[i].UpdatedAt = now, now
	}
	return true, s.tasks.CreateMany(ctx, tasks)
}
