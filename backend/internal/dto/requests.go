// Package dto holds the request and response bodies of the HTTP API.
package dto

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/models"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=80"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RoleRequest struct {
	Role models.Role `json:"role" binding:"required,oneof=admin manager member"`
}

type CreateProjectRequest struct {
	Name        string               `json:"name" binding:"required,min=2,max=120"`
	Description string               `json:"description" binding:"max=2000"`
	Status      string               `json:"status" binding:"omitempty,oneof=planned active completed"`
	MemberIDs   []primitive.ObjectID `json:"memberIds"`
}

// UpdateProjectRequest is a partial update: nil fields are left unchanged.
type UpdateProjectRequest struct {
	Name        *string               `json:"name" binding:"omitempty,min=2,max=120"`
	Description *string               `json:"description" binding:"omitempty,max=2000"`
	Status      *string               `json:"status" binding:"omitempty,oneof=planned active completed"`
	MemberIDs   *[]primitive.ObjectID `json:"memberIds"`
	// OwnerID transfers the project to another manager or admin; the previous owner stays on as a member.
	OwnerID *primitive.ObjectID `json:"ownerId"`
}

type CreateTaskRequest struct {
	Title       string              `json:"title" binding:"required,min=2,max=200"`
	Description string              `json:"description" binding:"max=5000"`
	Status      string              `json:"status" binding:"omitempty,oneof=todo in_progress done"`
	Priority    string              `json:"priority" binding:"omitempty,oneof=low medium high"`
	AssigneeID  *primitive.ObjectID `json:"assigneeId"`
	DueDate     string              `json:"dueDate" binding:"omitempty,datetime=2006-01-02"`
}

// UpdateTaskRequest is a partial update. An empty string clears assigneeId / dueDate;
// an absent field leaves it unchanged.
type UpdateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=2,max=200"`
	Description *string `json:"description" binding:"omitempty,max=5000"`
	Status      *string `json:"status" binding:"omitempty,oneof=todo in_progress done"`
	Priority    *string `json:"priority" binding:"omitempty,oneof=low medium high"`
	AssigneeID  *string `json:"assigneeId"`
	DueDate     *string `json:"dueDate"`
}

// OnlyStatus reports whether the request changes nothing but the status.
func (r UpdateTaskRequest) OnlyStatus() bool {
	return r.Title == nil && r.Description == nil && r.Priority == nil && r.AssigneeID == nil && r.DueDate == nil
}
