package controllers

import (
	"net/http"

	"classroom-attendance/go-backend/internal/middleware"
	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type PlatformController struct {
	platform   *services.PlatformService
	enrollment *services.EnrollmentService
}

func NewPlatformController(platform *services.PlatformService, enrollment *services.EnrollmentService) *PlatformController {
	return &PlatformController{platform: platform, enrollment: enrollment}
}

func (p *PlatformController) Dashboard(c *gin.Context) {
	claims := mustClaims(c)
	stats, err := p.platform.Dashboard(c.Request.Context(), claims.OrganizationID)
	write(c, stats, err, http.StatusOK)
}

func (p *PlatformController) ListTeachers(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListTeachers(c.Request.Context(), claims.OrganizationID)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateTeacher(c *gin.Context) {
	claims := mustClaims(c)
	var request models.TeacherCreateRequest
	if bind(c, &request) {
		teacher, err := p.platform.CreateTeacher(c.Request.Context(), claims.OrganizationID, request)
		write(c, teacher, err, http.StatusCreated)
	}
}

func (p *PlatformController) DeleteTeacher(c *gin.Context) {
	claims := mustClaims(c)
	err := p.platform.DeleteTeacher(c.Request.Context(), claims.OrganizationID, c.Param("id"))
	write(c, gin.H{"deleted": err == nil}, err, http.StatusOK)
}

func (p *PlatformController) ListStudents(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListStudents(c.Request.Context(), claims.OrganizationID)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateStudent(c *gin.Context) {
	claims := mustClaims(c)
	var request models.StudentCreateRequest
	if bind(c, &request) {
		student, err := p.platform.CreateStudent(c.Request.Context(), claims.OrganizationID, request)
		write(c, student, err, http.StatusCreated)
	}
}

func (p *PlatformController) DeleteStudent(c *gin.Context) {
	claims := mustClaims(c)
	err := p.platform.DeleteStudent(c.Request.Context(), claims.OrganizationID, c.Param("id"))
	write(c, gin.H{"deleted": err == nil}, err, http.StatusOK)
}

func (p *PlatformController) ListClasses(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListClasses(c.Request.Context(), claims.OrganizationID)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateClass(c *gin.Context) {
	claims := mustClaims(c)
	var request models.ClassCreateRequest
	if bind(c, &request) {
		row, err := p.platform.CreateClass(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) ListSubjects(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListSubjects(c.Request.Context(), claims.OrganizationID)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateSubject(c *gin.Context) {
	claims := mustClaims(c)
	var request models.SubjectCreateRequest
	if bind(c, &request) {
		row, err := p.platform.CreateSubject(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) ListAssignments(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListAssignments(c.Request.Context(), claims.OrganizationID)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateAssignment(c *gin.Context) {
	claims := mustClaims(c)
	var request models.AssignmentCreateRequest
	if bind(c, &request) {
		row, err := p.platform.CreateAssignment(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) ListCameras(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListCameras(c.Request.Context(), claims.OrganizationID)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateCamera(c *gin.Context) {
	claims := mustClaims(c)
	var request models.CameraCreateRequest
	if bind(c, &request) {
		row, err := p.platform.CreateCamera(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) ListSessions(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListAttendanceSessions(c.Request.Context(), claims.OrganizationID, claims.UserID, claims.Role)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) CreateSession(c *gin.Context) {
	claims := mustClaims(c)
	var request models.AttendanceSessionCreateRequest
	if bind(c, &request) {
		row, err := p.platform.CreateAttendanceSession(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) ListEvents(c *gin.Context) {
	claims := mustClaims(c)
	rows, err := p.platform.ListAttendanceEvents(c.Request.Context(), claims.OrganizationID, claims.UserID, claims.Role)
	write(c, rows, err, http.StatusOK)
}

func (p *PlatformController) SimulateRecognition(c *gin.Context) {
	claims := mustClaims(c)
	var request models.RecognitionSimulationRequest
	if bind(c, &request) {
		row, err := p.platform.SimulateRecognition(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) CorrectEvent(c *gin.Context) {
	claims := mustClaims(c)
	var request models.ManualCorrectionRequest
	if bind(c, &request) {
		row, err := p.platform.CorrectEvent(c.Request.Context(), claims.OrganizationID, claims.UserID, c.Param("id"), request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) EnrollStudent(c *gin.Context) {
	claims := mustClaims(c)
	var request models.EnrollmentRequest
	if bind(c, &request) {
		row, err := p.enrollment.EnrollStudent(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) DeleteStudentEnrollment(c *gin.Context) {
	claims := mustClaims(c)
	err := p.enrollment.DeleteStudentEnrollment(c.Request.Context(), claims.OrganizationID, c.Param("id"))
	write(c, gin.H{"deleted": err == nil}, err, http.StatusOK)
}

func (p *PlatformController) EnrollTeacher(c *gin.Context) {
	claims := mustClaims(c)
	var request models.EnrollmentRequest
	if bind(c, &request) {
		row, err := p.enrollment.EnrollTeacher(c.Request.Context(), claims.OrganizationID, request)
		write(c, row, err, http.StatusCreated)
	}
}

func (p *PlatformController) DeleteTeacherEnrollment(c *gin.Context) {
	claims := mustClaims(c)
	err := p.enrollment.DeleteTeacherEnrollment(c.Request.Context(), claims.OrganizationID, c.Param("id"))
	write(c, gin.H{"deleted": err == nil}, err, http.StatusOK)
}

func mustClaims(c *gin.Context) middlewareClaims {
	claims, _ := middleware.CurrentClaims(c)
	return middlewareClaims{UserID: claims.UserID, OrganizationID: claims.OrganizationID, Role: claims.Role}
}

type middlewareClaims struct {
	UserID         string
	OrganizationID string
	Role           string
}

func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func write(c *gin.Context, value any, err error, status int) {
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(status, value)
}
