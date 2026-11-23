package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"qt-devops-go-mongo/backend/internal/cache"
)

type HealthService interface {
	// Check returns an error when MongoDB cannot be reached.
	Check(ctx context.Context) error
	// CacheStatus reports "up", "down" or "disabled". It never affects overall health.
	CacheStatus(ctx context.Context) string
}

type healthService struct {
	client *mongo.Client
	cache  cache.Cache
}

func NewHealthService(client *mongo.Client, c cache.Cache) HealthService {
	return &healthService{client: client, cache: c}
}

func (s *healthService) CacheStatus(ctx context.Context) string {
	return s.cache.Status(ctx)
}

func (s *healthService) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.client.Ping(ctx, nil)
}
