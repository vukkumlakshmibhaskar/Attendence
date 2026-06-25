package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/worker"

	"github.com/gin-gonic/gin"
)

type FrameController struct {
	pool           *worker.Pool
	requestTimeout time.Duration
}

func NewFrameController(pool *worker.Pool, requestTimeout time.Duration) *FrameController {
	return &FrameController{pool: pool, requestTimeout: requestTimeout}
}

func (f *FrameController) Store(c *gin.Context) {
	var request models.FrameRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), f.requestTimeout)
	defer cancel()

	response, err := f.pool.Submit(ctx, request)
	if err != nil {
		if errors.Is(err, worker.ErrQueueFull) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "frame queue is full; client should skip this sample"})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (f *FrameController) Identify(c *gin.Context) {
	var request models.FrameRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), f.requestTimeout)
	defer cancel()

	response, err := f.pool.SubmitIdentify(ctx, request)
	if err != nil {
		if errors.Is(err, worker.ErrQueueFull) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "frame queue is full; client should skip this sample"})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}
