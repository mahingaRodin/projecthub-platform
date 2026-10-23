package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qt-devops-go-mongo/backend/internal/dto"
	"qt-devops-go-mongo/backend/internal/middleware"
	"qt-devops-go-mongo/backend/internal/service"
)

// StatsController serves the dashboard figures.
type StatsController struct {
	stats service.StatsService
}

func NewStatsController(stats service.StatsService) *StatsController {
	return &StatsController{stats: stats}
}

// Get handles GET /api/stats.
func (ctl *StatsController) Get(c *gin.Context) {
	res, err := ctl.stats.Get(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// HealthController reports API and database health.
type HealthController struct {
	health service.HealthService
}

func NewHealthController(health service.HealthService) *HealthController {
	return &HealthController{health: health}
}

// Check handles GET /api/health: 200 when MongoDB is reachable, 503 otherwise.
// The cache status is reported but never makes the API unhealthy.
func (ctl *HealthController) Check(c *gin.Context) {
	cacheStatus := ctl.health.CacheStatus(c.Request.Context())
	if err := ctl.health.Check(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.HealthResponse{Status: "error", Database: "down", Cache: cacheStatus})
		return
	}
	c.JSON(http.StatusOK, dto.HealthResponse{Status: "ok", Database: "up", Cache: cacheStatus})
}
