package server

import (
	"net/http"

	"classroom-attendance/go-backend/internal/config"
	"classroom-attendance/go-backend/internal/controllers"
	"classroom-attendance/go-backend/internal/middleware"
	"classroom-attendance/go-backend/internal/services"
	"classroom-attendance/go-backend/internal/worker"

	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	Config            config.Config
	AuthService       *services.AuthService
	PlatformService   *services.PlatformService
	EnrollmentService *services.EnrollmentService
	CacheService      *services.FaceCacheService
	Pool              *worker.Pool
}

func NewRouter(deps RouterDeps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware(deps.Config.CORSAllowedOrigin))

	healthController := controllers.NewHealthController(deps.CacheService, deps.Pool)
	faceCacheController := controllers.NewFaceCacheController(deps.CacheService)
	frameController := controllers.NewFrameController(deps.Pool, deps.Config.RequestTimeout)
	authController := controllers.NewAuthController(deps.AuthService)
	platformController := controllers.NewPlatformController(deps.PlatformService, deps.EnrollmentService)

	router.GET("/health", healthController.Show)
	router.GET("/api/auth/setup-status", authController.SetupStatus)
	router.POST("/api/auth/setup", authController.Setup)
	router.POST("/api/auth/login", authController.Login)

	protected := router.Group("/api")
	protected.Use(middleware.AuthRequired(deps.Config.JWTSecret))
	protected.GET("/auth/me", authController.Me)
	protected.GET("/dashboard", platformController.Dashboard)
	protected.GET("/teachers", platformController.ListTeachers)
	protected.GET("/students", platformController.ListStudents)
	protected.GET("/academics/classes", platformController.ListClasses)
	protected.GET("/academics/subjects", platformController.ListSubjects)
	protected.GET("/academics/assignments", platformController.ListAssignments)
	protected.GET("/attendance/sessions", platformController.ListSessions)
	protected.GET("/attendance/events", platformController.ListEvents)
	protected.GET("/cameras", platformController.ListCameras)
	protected.POST("/recognition/identify", frameController.Identify)
	protected.POST("/frames", frameController.Store)

	admin := protected.Group("")
	admin.Use(middleware.RequireRole("admin"))
	admin.POST("/teachers", platformController.CreateTeacher)
	admin.DELETE("/teachers/:id", platformController.DeleteTeacher)
	admin.POST("/students", platformController.CreateStudent)
	admin.DELETE("/students/:id", platformController.DeleteStudent)
	admin.POST("/academics/classes", platformController.CreateClass)
	admin.POST("/academics/subjects", platformController.CreateSubject)
	admin.POST("/academics/assignments", platformController.CreateAssignment)
	admin.POST("/attendance/sessions", platformController.CreateSession)
	admin.POST("/attendance/simulate", platformController.SimulateRecognition)
	admin.POST("/attendance/events/:id/corrections", platformController.CorrectEvent)
	admin.POST("/cameras", platformController.CreateCamera)
	admin.POST("/enrollments/students", platformController.EnrollStudent)
	admin.DELETE("/enrollments/students/:id", platformController.DeleteStudentEnrollment)
	admin.POST("/enrollments/teachers", platformController.EnrollTeacher)
	admin.DELETE("/enrollments/teachers/:id", platformController.DeleteTeacherEnrollment)
	admin.POST("/face-cache/reload", faceCacheController.Reload)

	return router
}

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
