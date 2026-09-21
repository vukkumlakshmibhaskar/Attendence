package controllers

import (
	"net/http"

	"classroom-attendance/go-backend/internal/middleware"
	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	auth *services.AuthService
}

func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

func (a *AuthController) SetupStatus(c *gin.Context) {
	complete, err := a.auth.SetupStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"setup_complete": complete})
}

func (a *AuthController) Setup(c *gin.Context) {
	var request models.SetupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, err := a.auth.Setup(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, response)
}

func (a *AuthController) Login(c *gin.Context) {
	var request models.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, err := a.auth.Login(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (a *AuthController) Me(c *gin.Context) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}
	response, err := a.auth.Me(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}
