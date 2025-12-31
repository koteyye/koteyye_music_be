package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "koteyye_music_be/docs" // Import docs for swagger

	"koteyye_music_be/internal/config"
	"koteyye_music_be/internal/handler"
	"koteyye_music_be/internal/middleware"
	"koteyye_music_be/internal/repository"
	"koteyye_music_be/internal/service"
	"koteyye_music_be/pkg/database"
	"koteyye_music_be/pkg/logger"
	"koteyye_music_be/pkg/migrations"
	"koteyye_music_be/pkg/minio"
)

// @title Koteyye Music API
// @version 1.0
// @description API for music streaming service.
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init("info"); err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}

	logger.Log.Info("Starting Music Service API", "port", cfg.ServerPort)

	// Initialize database connection
	db, err := database.NewDB(context.Background(), cfg.DBDSN)
	if err != nil {
		logger.Log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Log.Info("Database connected successfully")

	// Run database migrations
	logger.Log.Info("Running database migrations...")
	migrator := migrations.NewMigrator(db, logger.Log)
	if err := migrator.Up(context.Background()); err != nil {
		logger.Log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize MinIO client
	minioClient, err := minio.New(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
		logger.Log,
	)
	if err != nil {
		logger.Log.Error("Failed to initialize MinIO client", "error", err)
		os.Exit(1)
	}
	logger.Log.Info("MinIO connected successfully", "bucket", cfg.MinIOBucket)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	trackRepo := repository.NewTrackRepository(db)
	albumRepo := repository.NewAlbumRepository(db.Pool)

	// Initialize MinIO service
	minioService := minio.NewService(minioClient, cfg.MinIOEndpoint, cfg.MinIOUseSSL, logger.Log)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, logger.Log)
	userService := service.NewUserService(userRepo, minioClient, logger.Log)
	trackService := service.NewTrackService(trackRepo, albumRepo, minioClient, minioService, logger.Log)
	albumService := service.NewAlbumService(albumRepo, trackRepo, minioService)
	oauthService := service.NewOAuthService(
		userRepo,
		authService,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
		cfg.YandexClientID,
		cfg.YandexClientSecret,
		cfg.YandexRedirectURL,
		cfg.FrontendURL,
		logger.Log,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, logger.Log)
	userHandler := handler.NewUserHandler(userService, logger.Log)
	trackHandler := handler.NewTrackHandler(trackService, logger.Log)
	oauthHandler := handler.NewOAuthHandler(oauthService, logger.Log)
	adminHandler := handler.NewAdminHandler(trackService, albumService, logger.Log)
	albumHandler := handler.NewAlbumHandler(albumService, logger.Log)

	// Setup router
	router := setupRouter(authHandler, userHandler, trackHandler, oauthHandler, adminHandler, albumHandler, authService, userRepo)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Log.Info("Server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Log.Info("Server shutdown complete")
}

func setupRouter(authHandler *handler.AuthHandler, userHandler *handler.UserHandler, trackHandler *handler.TrackHandler, oauthHandler *handler.OAuthHandler, adminHandler *handler.AdminHandler, albumHandler *handler.AlbumHandler, authService *service.AuthService, userRepo *repository.UserRepository) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS)

	// Health check
	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
	r.Get("/health", healthHandler)
	r.Head("/health", healthHandler)

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Public auth routes
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/guest", authHandler.GuestLogin)

		// OAuth routes
		r.Get("/google/login", oauthHandler.OAuthLogin)
		r.Get("/google/callback", oauthHandler.OAuthCallback)
		r.Get("/yandex/login", oauthHandler.OAuthLogin)
		r.Get("/yandex/callback", oauthHandler.OAuthCallback)
	})

	// API routes with mixed authentication requirements
	r.Route("/api/tracks", func(r chi.Router) {
		// Public routes with optional authentication (lazy auth)
		r.With(middleware.OptionalAuthMiddleware(authService)).Get("/", trackHandler.ListTracks)
		r.With(middleware.OptionalAuthMiddleware(authService)).Get("/{id}", trackHandler.GetTrack)
		r.With(middleware.OptionalAuthMiddleware(authService)).Get("/{id}/stream", trackHandler.StreamTrack)
		r.With(middleware.OptionalAuthMiddleware(authService)).Head("/{id}/stream", trackHandler.StreamTrack) // Support HEAD for stream
		r.Get("/{id}/cover", trackHandler.GetTrackCover) // Public cover access
		r.Head("/{id}/cover", trackHandler.GetTrackCover) // Support HEAD for cover

		// Protected routes (require authentication including guests)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authService))
			r.Use(middleware.RequireAuth(userRepo))

			r.Get("/my", trackHandler.GetUserTracks)
			r.Post("/{id}/play", trackHandler.IncrementPlays)
			r.Post("/{id}/like", trackHandler.ToggleLike)
		})
	})

	// Album routes (public)
	r.Route("/api/albums", func(r chi.Router) {
		r.Get("/", albumHandler.GetAlbums)
		r.Get("/{id}", albumHandler.GetAlbumByID)
		r.Get("/{id}/info", albumHandler.GetAlbumInfo)
		r.Get("/{id}/cover", albumHandler.GetAlbumCover) // Public album cover access
		r.Head("/{id}/cover", albumHandler.GetAlbumCover) // Support HEAD for cover
	})

	// User profile routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))
		r.Use(middleware.RequireAuth(userRepo))

		r.Route("/api/users", func(r chi.Router) {
			r.Get("/me", userHandler.GetMe)
			r.Put("/me", userHandler.UpdateMe)
			r.Post("/me/avatar", userHandler.UploadAvatar)
			r.Delete("/me/avatar", userHandler.RemoveAvatar)
			r.Post("/player-state", userHandler.UpdatePlayerState) // Move player-state under /api/users/
		})
	})

	// Public avatar serving (no auth required)
	r.Get("/api/avatars/*", userHandler.GetAvatar)

	// Admin routes (require admin role)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))
		r.Use(middleware.RequireAdmin(userRepo))

		r.Route("/api/admin", func(r chi.Router) {
			// Album management (admin only)
			r.Route("/albums", func(r chi.Router) {
				r.Post("/", adminHandler.CreateAlbum)
				r.Delete("/{id}", adminHandler.DeleteAlbum)
				r.Post("/{id}/tracks", adminHandler.AddTrackToAlbum)
			})

			// Track management (admin only)
			r.Route("/tracks", func(r chi.Router) {
				r.Post("/upload", trackHandler.UploadTrack)
				r.Delete("/{id}", adminHandler.DeleteTrack)
			})
		})
	})

	return r
}
