package dto

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"qt-devops-go-mongo/backend/internal/models"
)

type AuthResponse struct {
	Token     string            `json:"token"`
	ExpiresIn int               `json:"expiresIn"`
	User      models.PublicUser `json:"user"`
}

type MeResponse struct {
	User models.PublicUser `json:"user"`
}

type RoleResponse struct {
	ID   primitive.ObjectID `json:"id"`
	Role models.Role        `json:"role"`
}

type ProjectResponse struct {
	models.Project
	Owner     *models.PublicUser  `json:"owner,omitempty"`
	Members   []models.PublicUser `json:"members"`
	TaskCount int                 `json:"taskCount"`
	DoneCount int                 `json:"doneCount"`
	CanManage bool                `json:"canManage"`
}

type MyTask struct {
	models.Task
	ProjectName string `json:"projectName"`
}

type StatsResponse struct {
	Projects         int            `json:"projects"`
	ProjectsByStatus map[string]int `json:"projectsByStatus"`
	TasksByStatus    map[string]int `json:"tasksByStatus"`
	OverdueTasks     int64          `json:"overdueTasks"`
	MyOpenTasks      []MyTask       `json:"myOpenTasks"`
}

// SeedUser describes a demo account. Passwords are documented in the README, never returned by the API.
type SeedUser struct {
	Name  string      `json:"name"`
	Email string      `json:"email"`
	Role  models.Role `json:"role"`
}

type SeedResponse struct {
	Message        string     `json:"message"`
	UsersCreated   int        `json:"usersCreated"`
	ProjectCreated bool       `json:"projectCreated"`
	Users          []SeedUser `json:"users"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Cache    string `json:"cache"` // up | down | disabled — informational, the cache is optional
}
