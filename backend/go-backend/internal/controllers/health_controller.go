package controllers

import (
	"net/http"

	"classroom-attendance/go-backend/internal/services"
	"classroom-attendance/go-backend/internal/worker"

	"github.com/gin-gonic/gin"
)

type HealthController struct {
	cacheService *services.FaceCacheService
	pool         *worker.Pool
}

func NewHealthController(cacheService *services.FaceCacheService, pool *worker.Pool) *HealthController {
	return &HealthController{cacheService: cacheService, pool: pool}
}

func (h *HealthController) Show(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":          "ok",
		"face_cache_size": h.cacheService.Size(),
		"worker_count":    h.pool.WorkerCount(),
	})
}
