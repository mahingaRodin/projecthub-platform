package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qt-devops-go-mongo/backend/internal/cache"
	"qt-devops-go-mongo/backend/internal/config"
	"qt-devops-go-mongo/backend/internal/controller"
	"qt-devops-go-mongo/backend/internal/database"
	"qt-devops-go-mongo/backend/internal/repository"
	"qt-devops-go-mongo/backend/internal/router"
	"qt-devops-go-mongo/backend/internal/service"
)

func main() {
	cfg := config.Load()

	client, db, err := database.Connect(cfg.MongoURI, cfg.MongoDB, cfg.MongoConnectTimeout)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	defer client.Disconnect(context.Background())

	if err := database.EnsureIndexes(context.Background(), db); err != nil {
		log.Fatalf("mongodb indexes: %v", err)
	}

	// Optional cache: the API works without it (see internal/cache).
	var appCache cache.Cache = cache.Noop{}
	if cfg.CacheEnabled {
		appCache = cache.NewValkey(cfg.ValkeyAddr, cfg.ValkeyPassword, cfg.ValkeyDB)
	}
	defer appCache.Close()

	// Wiring: repository -> service -> controller -> router.
	users := repository.NewUserRepository(db)
	projects := repository.NewProjectRepository(db)
	tasks := repository.NewTaskRepository(db)

	authService := service.NewAuthService(users, appCache, cfg)
	seedService := service.NewSeedService(cfg.SeedEnabled, users, projects, tasks, appCache)
	if cfg.SeedEnabled {
		seedOnFirstStart(users, seedService)
	}
	controllers := router.Controllers{
		Auth:     controller.NewAuthController(authService),
		Users:    controller.NewUserController(service.NewUserService(users, projects, tasks, appCache)),
		Projects: controller.NewProjectController(service.NewProjectService(projects, tasks, users, appCache)),
		Tasks:    controller.NewTaskController(service.NewTaskService(tasks, projects, appCache)),
		Stats:    controller.NewStatsController(service.NewStatsService(projects, tasks, appCache, cfg.CacheStatsTTL)),
		Health:   controller.NewHealthController(service.NewHealthService(client, appCache)),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(cfg, authService, controllers),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("ProjectHub API listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM (docker stop, kubectl delete pod).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// seedOnFirstStart creates the test accounts and a sample project when the database has no users yet,
// so a fresh deployment is usable immediately. Later restarts do nothing. There is no seed endpoint.
func seedOnFirstStart(users *repository.UserRepository, seed service.SeedService) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	n, err := users.Count(ctx)
	if err != nil {
		log.Printf("WARN initial seed skipped: %v", err)
		return
	}
	if n > 0 {
		return
	}
	res, err := seed.Seed(ctx)
	if err != nil {
		log.Printf("WARN initial seed failed: %v", err)
		return
	}
	log.Printf("first start: created %d test accounts (project created: %t) — see README", res.UsersCreated, res.ProjectCreated)
}
