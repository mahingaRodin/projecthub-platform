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

type ProjectService interface {
	// List returns every project for admins, and owned or joined projects for everyone else.
	List(ctx context.Context, actor *models.User) ([]dto.ProjectResponse, error)
	Get(ctx context.Context, actor *models.User, id primitive.ObjectID) (*dto.ProjectResponse, error)
	Create(ctx context.Context, actor *models.User, req dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	// Update and Delete are allowed for the project owner and admins. Update can also transfer ownership.
	Update(ctx context.Context, actor *models.User, id primitive.ObjectID, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	Delete(ctx context.Context, actor *models.User, id primitive.ObjectID) error
}

type projectService struct {
	projects *repository.ProjectRepository
	tasks    *repository.TaskRepository
	users    *repository.UserRepository
	cache    cache.Cache
}

func NewProjectService(projects *repository.ProjectRepository, tasks *repository.TaskRepository, users *repository.UserRepository, c cache.Cache) ProjectService {
	return &projectService{projects: projects, tasks: tasks, users: users, cache: c}
}

func (s *projectService) List(ctx context.Context, actor *models.User) ([]dto.ProjectResponse, error) {
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
	return s.toResponses(ctx, actor, projects)
}

func (s *projectService) Get(ctx context.Context, actor *models.User, id primitive.ObjectID) (*dto.ProjectResponse, error) {
	p, err := loadVisibleProject(ctx, s.projects, actor, id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(ctx, actor, p)
}

func (s *projectService) Create(ctx context.Context, actor *models.User, req dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	members := cleanMembers(req.MemberIDs, actor.ID)
	if err := s.checkUsersExist(ctx, members); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	p := &models.Project{
		ID:          primitive.NewObjectID(),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Status:      orDefault(req.Status, "planned"),
		OwnerID:     actor.ID,
		MemberIDs:   members,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.projects.Create(ctx, p); err != nil {
		return nil, err
	}
	s.cache.Bump(ctx, cache.StatsGroup)
	return s.toResponse(ctx, actor, p)
}

func (s *projectService) Update(ctx context.Context, actor *models.User, id primitive.ObjectID, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	p, err := loadVisibleProject(ctx, s.projects, actor, id)
	if err != nil {
		return nil, err
	}
	if !canManage(actor, p) {
		return nil, apperror.Forbidden("only the project owner or an admin can edit this project")
	}
	if req.Name != nil {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		p.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.MemberIDs != nil {
		members := cleanMembers(*req.MemberIDs, p.OwnerID)
		if err := s.checkUsersExist(ctx, members); err != nil {
			return nil, err
		}
		p.MemberIDs = members
	}
	if req.OwnerID != nil && *req.OwnerID != p.OwnerID {
		if err := s.transferOwnership(ctx, p, *req.OwnerID); err != nil {
			return nil, err
		}
	}
	p.UpdatedAt = time.Now().UTC()
	if err := s.projects.Replace(ctx, p); err != nil {
		return nil, err
	}
	s.cache.Bump(ctx, cache.StatsGroup)
	return s.toResponse(ctx, actor, p)
}

func (s *projectService) Delete(ctx context.Context, actor *models.User, id primitive.ObjectID) error {
	p, err := loadVisibleProject(ctx, s.projects, actor, id)
	if err != nil {
		return err
	}
	if !canManage(actor, p) {
		return apperror.Forbidden("only the project owner or an admin can delete this project")
	}
	if err := s.tasks.DeleteByProject(ctx, p.ID); err != nil {
		return err
	}
	if err := s.projects.Delete(ctx, p.ID); err != nil {
		return err
	}
	s.cache.Bump(ctx, cache.StatsGroup)
	return nil
}

// ---- helpers ------------------------------------------------------------

// transferOwnership makes newOwner the owner of p. Only managers and admins can own projects.
// The previous owner stays on the team as a member so they keep access and their task assignments.
func (s *projectService) transferOwnership(ctx context.Context, p *models.Project, newOwner primitive.ObjectID) error {
	u, err := s.users.FindByID(ctx, newOwner)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.BadRequest("the new owner does not exist")
	}
	if err != nil {
		return err
	}
	if u.Role != models.RoleManager && u.Role != models.RoleAdmin {
		return apperror.BadRequest("only a manager or an admin can own a project")
	}
	previous := p.OwnerID
	p.OwnerID = newOwner
	p.MemberIDs = cleanMembers(append(p.MemberIDs, previous), newOwner)
	return nil
}

func (s *projectService) toResponse(ctx context.Context, actor *models.User, p *models.Project) (*dto.ProjectResponse, error) {
	out, err := s.toResponses(ctx, actor, []models.Project{*p})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

// toResponses adds owner, members and task counts using one query per collection.
func (s *projectService) toResponses(ctx context.Context, actor *models.User, projects []models.Project) ([]dto.ProjectResponse, error) {
	var userIDs, projectIDs []primitive.ObjectID
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
		userIDs = append(userIDs, p.OwnerID)
		userIDs = append(userIDs, p.MemberIDs...)
	}
	users, err := s.users.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	people := make(map[primitive.ObjectID]models.PublicUser, len(users))
	for _, u := range users {
		people[u.ID] = u.Public()
	}
	counts, err := s.tasks.CountsByProject(ctx, projectIDs)
	if err != nil {
		return nil, err
	}

	out := make([]dto.ProjectResponse, 0, len(projects))
	for _, p := range projects {
		r := dto.ProjectResponse{Project: p, Members: []models.PublicUser{}, CanManage: canManage(actor, &p)}
		if r.MemberIDs == nil {
			r.MemberIDs = []primitive.ObjectID{}
		}
		if owner, ok := people[p.OwnerID]; ok {
			r.Owner = &owner
		}
		for _, id := range p.MemberIDs {
			if m, ok := people[id]; ok {
				r.Members = append(r.Members, m)
			}
		}
		r.TaskCount, r.DoneCount = counts[p.ID].Total, counts[p.ID].Done
		out = append(out, r)
	}
	return out, nil
}

func (s *projectService) checkUsersExist(ctx context.Context, ids []primitive.ObjectID) error {
	if len(ids) == 0 {
		return nil
	}
	n, err := s.users.CountByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if n != int64(len(ids)) {
		return apperror.BadRequest("one or more memberIds do not exist")
	}
	return nil
}

// cleanMembers removes duplicates and the owner from a member list.
func cleanMembers(ids []primitive.ObjectID, owner primitive.ObjectID) []primitive.ObjectID {
	out := []primitive.ObjectID{}
	for _, id := range ids {
		if id != owner && !containsID(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
