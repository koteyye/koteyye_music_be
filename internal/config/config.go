package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN          string
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	JWTSecret      string
	ServerPort     string
	// OAuth Google
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	// OAuth Yandex
	YandexClientID     string
	YandexClientSecret string
	YandexRedirectURL  string
	// Frontend
	FrontendURL string
}

func Load() (*Config, error) {
	// Загружаем .env файл, если он существует
	_ = godotenv.Load()

	// Собираем DSN из отдельных переменных или используем готовую
	dbDSN := getEnv("DB_DSN", "")
	if dbDSN == "" {
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbUser := getEnv("DB_USER", "postgres_user")
		dbPassword := getEnv("DB_PASSWORD", "postgres_pass")
		dbName := getEnv("DB_NAME", "music_service")
		dbDSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	cfg := &Config{
		DBDSN:          dbDSN,
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    getEnv("MINIO_BUCKET", "music-files"),
		MinIOUseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		JWTSecret:      getEnv("JWT_SECRET", "default-secret-key-change-in-production"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		// OAuth Google
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		// OAuth Yandex
		YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
		YandexRedirectURL:  getEnv("YANDEX_REDIRECT_URL", "http://localhost:8080/api/auth/yandex/callback"),
		// Frontend
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
