package controllers

import (
	"net/http"

	"classroom-attendance/go-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type FaceCacheController struct {
	cacheService *services.FaceCacheService
}

func NewFaceCacheController(cacheService *services.FaceCacheService) *FaceCacheController {
	return &FaceCacheController{cacheService: cacheService}
}

func (f *FaceCacheController) Reload(c *gin.Context) {
	size, err := f.cacheService.Reload(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          "reloaded",
		"face_cache_size": size,
	})
}
