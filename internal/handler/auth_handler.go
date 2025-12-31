package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/service"
	"koteyye_music_be/pkg/logger"
)

type AuthHandler struct {
	authService *service.AuthService
	logger      *slog.Logger
}

func NewAuthHandler(authService *service.AuthService, log *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      log,
	}
}

// Register handles user registration
// @Summary User Registration
// @Tags auth
// @Accept json
// @Produce json
// @Param input body models.RegisterRequest true "Registration data"
// @Success 201 {object} models.AuthResponse "User successfully registered"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 409 {object} map[string]string "Conflict - user already exists"
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req models.RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to parse request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if req.Email == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Email is required")
		return
	}
	if len(req.Password) < 6 {
		sendErrorResponse(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Call auth service
	response, err := h.authService.Register(ctx, &req)
	if err != nil {
		if errors.Is(err, errors.New("user with this email already exists")) {
			sendErrorResponse(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.Error("Registration failed", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	sendJSONResponse(w, http.StatusCreated, response)
}

// Login handles user login
// @Summary User Login
// @Tags auth
// @Accept json
// @Produce json
// @Param input body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.AuthResponse "User successfully logged in"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid credentials"
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req models.LoginRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to parse request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate request
	if req.Email == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Email is required")
		return
	}
	if req.Password == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Password is required")
		return
	}

	// Call auth service
	response, err := h.authService.Login(ctx, &req)
	if err != nil {
		if errors.Is(err, errors.New("invalid email or password")) {
			sendErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}
		h.logger.Error("Login failed", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Login failed")
		return
	}

	sendJSONResponse(w, http.StatusOK, response)
}

// GuestLogin handles guest user login (no registration required)
// @Summary Guest Login
// @Description Creates a temporary guest user without registration. Guest users can browse tracks, like, and play music. When they login via OAuth, the guest account will be promoted to a registered user.
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} models.GuestResponse "Guest successfully logged in"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/auth/guest [post]
func (h *AuthHandler) GuestLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Info("GuestLogin endpoint called")

	// Call auth service to create guest user
	response, err := h.authService.GuestLogin(ctx)
	if err != nil {
		h.logger.Error("Guest login failed", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to create guest session")
		return
	}

	sendJSONResponse(w, http.StatusOK, response)
}

// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Log.Error("Failed to encode JSON response", "error", err)
	}
}

// sendErrorResponse sends an error JSON response
func sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": message,
	}

	json.NewEncoder(w).Encode(response)
}
