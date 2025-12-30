package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"koteyye_music_be/internal/middleware"
	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/service"
)

type UserHandler struct {
	userService *service.UserService
	logger      *slog.Logger
}

func NewUserHandler(userService *service.UserService, log *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      log,
	}
}

// GetMe retrieves current user profile
// @Summary Get Current User Profile
// @Security BearerAuth
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} models.UserProfileResponse "User profile data"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/users/me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user profile
	profile, err := h.userService.GetUserProfile(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user profile", "user_id", userID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get user profile")
		return
	}

	sendJSONResponse(w, http.StatusOK, profile)
}

// UpdateMe updates current user profile
// @Summary Update Current User Profile
// @Security BearerAuth
// @Tags users
// @Accept json
// @Produce json
// @Param input body models.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} models.UserProfileResponse "Updated user profile"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/users/me [put]
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req models.UpdateProfileRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to parse request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Update user profile
	profile, err := h.userService.UpdateUserProfile(ctx, userID, &req)
	if err != nil {
		h.logger.Error("Failed to update user profile", "user_id", userID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to update user profile")
		return
	}

	sendJSONResponse(w, http.StatusOK, profile)
}

// UploadAvatar handles avatar file upload
// @Summary Upload User Avatar
// @Security BearerAuth
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Avatar image file (jpg, png, gif, webp, max 5MB)"
// @Success 200 {object} models.UserProfileResponse "Updated user profile with new avatar"
// @Failure 400 {object} map[string]string "Bad request - invalid file"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 413 {object} map[string]string "File too large"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/users/me/avatar [post]
func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse multipart form (max 10MB in memory)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.logger.Error("Failed to parse multipart form", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("avatar")
	if err != nil {
		h.logger.Error("Failed to get avatar file", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Avatar file is required")
		return
	}
	defer file.Close()

	// Upload avatar
	profile, err := h.userService.UploadAvatar(ctx, userID, file, header)
	if err != nil {
		h.logger.Error("Failed to upload avatar", "user_id", userID, "error", err)
		if strings.Contains(err.Error(), "file too large") || strings.Contains(err.Error(), "unsupported file type") {
			sendErrorResponse(w, http.StatusBadRequest, err.Error())
		} else {
			sendErrorResponse(w, http.StatusInternalServerError, "Failed to upload avatar")
		}
		return
	}

	sendJSONResponse(w, http.StatusOK, profile)
}

// RemoveAvatar handles avatar removal
// @Summary Remove User Avatar
// @Security BearerAuth
// @Tags users
// @Produce json
// @Success 200 {object} models.UserProfileResponse "Updated user profile without avatar"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/users/me/avatar [delete]
func (h *UserHandler) RemoveAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Remove avatar
	profile, err := h.userService.RemoveAvatar(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to remove avatar", "user_id", userID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to remove avatar")
		return
	}

	sendJSONResponse(w, http.StatusOK, profile)
}

// GetAvatar serves avatar files from MinIO
// @Summary Get User Avatar
// @Tags users
// @Param key path string true "Avatar key" Example(avatars/123/uuid.jpg)
// @Success 200 {file} binary "Avatar image"
// @Failure 404 {object} map[string]string "Avatar not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/avatars/{key} [get]
func (h *UserHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract avatar key from URL path
	avatarKey := r.URL.Path[len("/api/avatars/"):]
	if avatarKey == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Avatar key is required")
		return
	}

	// Get avatar stream
	stream, contentType, size, err := h.userService.GetAvatarStream(ctx, avatarKey)
	if err != nil {
		h.logger.Error("Failed to get avatar", "avatar_key", avatarKey, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Avatar not found")
		return
	}
	defer stream.Close()

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day

	// Copy stream to response
	if _, err := io.Copy(w, stream); err != nil {
		h.logger.Error("Failed to stream avatar", "avatar_key", avatarKey, "error", err)
	}
}

// UpdatePlayerState updates user's player state (track, position, volume)
// @Summary Update Player State
// @Security BearerAuth
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.PlayerStateRequest true "Player state data"
// @Success 200 "Player state updated successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input data"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/user/player-state [post]
func (h *UserHandler) UpdatePlayerState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request body
	var req models.PlayerStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate input
	if req.TrackID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(req.TrackID); err != nil {
		h.logger.Warn("Invalid track ID format", "track_id", req.TrackID, "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Track ID must be a valid UUID")
		return
	}

	if req.Position < 0 {
		sendErrorResponse(w, http.StatusBadRequest, "Position must be non-negative")
		return
	}
	if req.Volume < 0 || req.Volume > 100 {
		sendErrorResponse(w, http.StatusBadRequest, "Volume must be between 0 and 100")
		return
	}

	// Update player state
	err := h.userService.UpdatePlayerState(ctx, userID, req.TrackID, req.Position, req.Volume)
	if err != nil {
		h.logger.Error("Failed to update player state", "user_id", userID, "track_id", req.TrackID, "error", err)

		// Return specific error messages for better UX
		if strings.Contains(err.Error(), "track not found") {
			sendErrorResponse(w, http.StatusNotFound, "Track not found")
			return
		}
		if strings.Contains(err.Error(), "user not found") {
			sendErrorResponse(w, http.StatusNotFound, "User not found")
			return
		}

		sendErrorResponse(w, http.StatusInternalServerError, "Failed to update player state")
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Player state updated successfully"}`))
}
