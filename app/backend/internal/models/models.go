package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Role string

const (
	RoleAdmin   Role = "admin"   // manages users and every project
	RoleManager Role = "manager" // creates projects and manages the ones they own
	RoleMember  Role = "member"  // works on tasks in projects they belong to
)

type User struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Role         Role               `bson:"role" json:"role"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

type Project struct {
	ID          primitive.ObjectID   `bson:"_id" json:"id"`
	Name        string               `bson:"name" json:"name"`
	Description string               `bson:"description" json:"description"`
	Status      string               `bson:"status" json:"status"` // planned | active | completed
	OwnerID     primitive.ObjectID   `bson:"ownerId" json:"ownerId"`
	MemberIDs   []primitive.ObjectID `bson:"memberIds" json:"memberIds"`
	CreatedAt   time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type Task struct {
	ID          primitive.ObjectID  `bson:"_id" json:"id"`
	ProjectID   primitive.ObjectID  `bson:"projectId" json:"projectId"`
	Title       string              `bson:"title" json:"title"`
	Description string              `bson:"description" json:"description"`
	Status      string              `bson:"status" json:"status"`     // todo | in_progress | done
	Priority    string              `bson:"priority" json:"priority"` // low | medium | high
	AssigneeID  *primitive.ObjectID `bson:"assigneeId,omitempty" json:"assigneeId"`
	DueDate     *time.Time          `bson:"dueDate,omitempty" json:"dueDate"`
	CreatedBy   primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt"`
}

// PublicUser is the safe projection of a user embedded in other responses.
type PublicUser struct {
	ID    primitive.ObjectID `json:"id"`
	Name  string             `json:"name"`
	Email string             `json:"email"`
	Role  Role               `json:"role"`
}

func (u User) Public() PublicUser {
	return PublicUser{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role}
}
