package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/repositories"
	"classroom-attendance/go-backend/internal/security"
)

type AuthService struct {
	repository *repositories.PlatformRepository
	jwtSecret  string
}

func NewAuthService(repository *repositories.PlatformRepository, jwtSecret string) *AuthService {
	return &AuthService{repository: repository, jwtSecret: jwtSecret}
}

func (s *AuthService) SetupStatus(ctx context.Context) (bool, error) {
	return s.repository.SetupComplete(ctx)
}

func (s *AuthService) Setup(ctx context.Context, request models.SetupRequest) (models.AuthResponse, error) {
	exists, err := s.repository.SetupComplete(ctx)
	if err != nil {
		return models.AuthResponse{}, err
	}
	if exists {
		return models.AuthResponse{}, errors.New("first-time setup is already complete")
	}
	if strings.TrimSpace(request.OrganizationName) == "" || strings.TrimSpace(request.Email) == "" || len(request.Password) < 8 {
		return models.AuthResponse{}, errors.New("organization, email, and password with at least 8 characters are required")
	}

	hash, err := security.HashPassword(request.Password)
	if err != nil {
		return models.AuthResponse{}, err
	}

	org := models.Organization{
		Name: request.OrganizationName,
		Slug: slugify(request.OrganizationName),
	}
	admin := models.User{
		Name:         defaultString(request.AdminName, "Administrator"),
		Email:        strings.ToLower(strings.TrimSpace(request.Email)),
		PasswordHash: hash,
		Role:         "admin",
		Status:       "active",
	}

	org, admin, err = s.repository.CreateOrganizationAndAdmin(ctx, org, admin)
	if err != nil {
		return models.AuthResponse{}, err
	}

	token, err := s.token(admin)
	if err != nil {
		return models.AuthResponse{}, err
	}
	return models.AuthResponse{Token: token, User: admin, Organization: org}, nil
}

func (s *AuthService) Login(ctx context.Context, request models.LoginRequest) (models.AuthResponse, error) {
	user, err := s.repository.FindUserByEmail(ctx, strings.ToLower(strings.TrimSpace(request.Email)))
	if err != nil {
		return models.AuthResponse{}, errors.New("invalid email or password")
	}
	if user.Status != "active" {
		return models.AuthResponse{}, errors.New("account is inactive")
	}
	if !security.CheckPassword(user.PasswordHash, request.Password) {
		return models.AuthResponse{}, errors.New("invalid email or password")
	}

	org, err := s.repository.FindOrganizationByID(ctx, user.OrganizationID)
	if err != nil {
		return models.AuthResponse{}, err
	}
	token, err := s.token(user)
	if err != nil {
		return models.AuthResponse{}, err
	}
	return models.AuthResponse{Token: token, User: user, Organization: org}, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (models.AuthResponse, error) {
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return models.AuthResponse{}, err
	}
	org, err := s.repository.FindOrganizationByID(ctx, user.OrganizationID)
	if err != nil {
		return models.AuthResponse{}, err
	}
	return models.AuthResponse{User: user, Organization: org}, nil
}

func (s *AuthService) token(user models.User) (string, error) {
	return security.GenerateToken(s.jwtSecret, security.Claims{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		Role:           user.Role,
		Email:          user.Email,
	}, 12*time.Hour)
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	if value == "" {
		return "organization"
	}
	return value
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
