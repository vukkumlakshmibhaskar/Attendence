package database

import (
	"context"
	"database/sql"
	"time"

	"classroom-attendance/go-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

func New(ctx context.Context, databaseDSN string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseDSN), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}

	return &Store{db: db, sqlDB: sqlDB}, nil
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) AutoMigrate(ctx context.Context) error {
	if err := s.db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.AcademicClass{},
		&models.Subject{},
		&models.TeacherAssignment{},
		&models.Student{},
		&models.FaceEnrollment{},
		&models.TeacherFaceEnrollment{},
		&models.Camera{},
		&models.AttendanceSession{},
		&models.AttendanceEvent{},
		&models.ManualCorrection{},
		&models.AuditLog{},
		&models.AttendanceLog{},
		&models.CognitiveSnapshot{},
		&models.UnknownFaceEvent{},
	)
}

func (s *Store) Close() {
	if s.sqlDB != nil {
		_ = s.sqlDB.Close()
	}
}
