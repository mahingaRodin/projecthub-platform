// Package service contains the business logic and authorization rules. Each service is exposed as an
// interface (e.g. ProjectService) with an unexported implementation created by its New... function.
package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/apperror"
	"qt-devops-go-mongo/backend/internal/models"
	"qt-devops-go-mongo/backend/internal/repository"
)

// ---- authorization rules ------------------------------------------------

func canView(u *models.User, p *models.Project) bool {
	return u.Role == models.RoleAdmin || p.OwnerID == u.ID || containsID(p.MemberIDs, u.ID)
}

func canManage(u *models.User, p *models.Project) bool {
	return u.Role == models.RoleAdmin || p.OwnerID == u.ID
}

func isParticipant(p *models.Project, id primitive.ObjectID) bool {
	return p.OwnerID == id || containsID(p.MemberIDs, id)
}

// loadVisibleProject returns the project if the actor may see it. Projects the actor cannot see are
// reported as 404 so their existence is not leaked.
func loadVisibleProject(ctx context.Context, projects *repository.ProjectRepository, actor *models.User, id primitive.ObjectID) (*models.Project, error) {
	p, err := projects.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && !canView(actor, p)) {
		return nil, apperror.NotFound("project not found")
	}
	return p, err
}

func containsID(ids []primitive.ObjectID, id primitive.ObjectID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
