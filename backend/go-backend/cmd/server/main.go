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

	"classroom-attendance/go-backend/internal/cognitive"
	"classroom-attendance/go-backend/internal/config"
	"classroom-attendance/go-backend/internal/database"
	"classroom-attendance/go-backend/internal/recognition"
	"classroom-attendance/go-backend/internal/repositories"
	"classroom-attendance/go-backend/internal/server"
	"classroom-attendance/go-backend/internal/services"
	"classroom-attendance/go-backend/internal/worker"
)

func main() {
	cfg := config.Load()

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := database.New(rootCtx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer store.Close()

	if err := store.AutoMigrate(rootCtx); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	attendanceRepository := repositories.NewAttendanceRepository(store.DB())
	platformRepository := repositories.NewPlatformRepository(store.DB())
	vectors, err := attendanceRepository.LoadFaceVectors(rootCtx)
	if err != nil {
		log.Printf("face cache load warning: %v", err)
		vectors = nil
	}
	cache := recognition.NewFaceCache(vectors)
	log.Printf("loaded %d face embeddings into memory", cache.Size())

	recognizer, err := recognition.NewRecognizer(cfg.FaceRecognizerMode, cfg.DlibModelDir)
	if err != nil {
		log.Fatalf("face recognizer init failed: %v", err)
	}
	defer recognizer.Close()

	cognitiveClient := cognitive.NewClient(cfg.CognitiveServiceURL, cfg.RequestTimeout)
	if err := cognitiveClient.Health(rootCtx); err != nil {
		log.Printf("cognitive service health warning: %v", err)
	}

	verifierModel, err := recognition.LoadPairVerifierModel(cfg.FaceVerifierModelPath)
	if err != nil {
		log.Printf("face verifier model warning: %v", err)
	}
	if verifierModel != nil {
		log.Printf("loaded face verifier model %s with threshold %.3f", cfg.FaceVerifierModelPath, verifierModel.Threshold)
	}

	attendanceService := services.NewAttendanceService(
		attendanceRepository,
		cache,
		recognition.NewMatcher(cfg.MatchThreshold, verifierModel),
		recognizer,
		cognitiveClient,
	)
	faceCacheService := services.NewFaceCacheService(attendanceRepository, cache)
	authService := services.NewAuthService(platformRepository, cfg.JWTSecret)
	platformService := services.NewPlatformService(platformRepository)
	enrollmentService := services.NewEnrollmentService(
		platformRepository,
		services.NewEmbeddingService(cfg.FaceEmbeddingModelPath),
		faceCacheService,
	)

	pool := worker.NewPool(attendanceService, cfg.MaxFrameWorkers, cfg.FrameQueueSize)
	pool.Start(rootCtx)

	router := server.NewRouter(server.RouterDeps{
		Config:            cfg,
		AuthService:       authService,
		PlatformService:   platformService,
		EnrollmentService: enrollmentService,
		CacheService:      faceCacheService,
		Pool:              pool,
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("attendance API listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-rootCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown failed: %v", err)
	}
}
