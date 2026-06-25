package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr               string
	DatabaseDSN            string
	DBConnection           string
	CognitiveServiceURL    string
	MaxFrameWorkers        int
	FrameQueueSize         int
	MatchThreshold         float64
	FaceRecognizerMode     string
	DlibModelDir           string
	CORSAllowedOrigin      string
	RequestTimeout         time.Duration
	JWTSecret              string
	FaceEmbeddingModelPath string
	FaceVerifierModelPath  string
}

func Load() Config {
	loadDotEnv(".env")

	databaseDSN, dbConnection := buildDatabaseDSN()
	return Config{
		HTTPAddr:               getenv("HTTP_ADDR", ":8080"),
		DatabaseDSN:            databaseDSN,
		DBConnection:           dbConnection,
		CognitiveServiceURL:    getenv("COGNITIVE_SERVICE_URL", "http://localhost:8001"),
		MaxFrameWorkers:        getenvInt("MAX_FRAME_WORKERS", 4),
		FrameQueueSize:         getenvInt("FRAME_QUEUE_SIZE", 16),
		MatchThreshold:         getenvFloat("MATCH_THRESHOLD", 0.55),
		FaceRecognizerMode:     getenv("FACE_RECOGNIZER_MODE", "mock"),
		DlibModelDir:           getenv("DLIB_MODEL_DIR", "models/dlib"),
		CORSAllowedOrigin:      getenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173"),
		RequestTimeout:         time.Duration(getenvInt("REQUEST_TIMEOUT_SECONDS", 12)) * time.Second,
		JWTSecret:              getenv("JWT_SECRET", "change-this-local-secret"),
		FaceEmbeddingModelPath: getenv("FACE_EMBEDDING_MODEL_PATH", "models/face_embedding.onnx"),
		FaceVerifierModelPath:  getenv("FACE_VERIFIER_MODEL_PATH", "models/face_verifier_lfw.json"),
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func buildDatabaseDSN() (string, string) {
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		return databaseURL, "url"
	}

	connection := getenv("DB_CONNECTION", "pgsql")
	host := getenv("DB_HOST", "127.0.0.1")
	port := getenv("DB_PORT", "5432")
	database := getenv("DB_DATABASE", "attendance")
	username := getenv("DB_USERNAME", "attendance")
	password := os.Getenv("DB_PASSWORD")

	if connection != "pgsql" && connection != "postgres" && connection != "postgresql" {
		connection = "pgsql"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
		host,
		username,
		password,
		database,
		port,
	)
	return dsn, connection
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
