package handler

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"koteyye_music_be/internal/middleware"
	"koteyye_music_be/internal/service"

	"github.com/go-chi/chi/v5"
)

type TrackHandler struct {
	trackService *service.TrackService
	logger       *slog.Logger
}

func NewTrackHandler(trackService *service.TrackService, log *slog.Logger) *TrackHandler {
	return &TrackHandler{
		trackService: trackService,
		logger:       log,
	}
}

// UploadTrack handles track upload
// @Summary Upload Track (Admin)
// @Security BearerAuth
// @Tags admin
// @Accept mpfd
// @Produce json
// @Param title formData string true "Track Title" Example(Bohemian Rhapsody)
// @Param artist formData string false "Artist Name" Example(Queen)
// @Param album formData string false "Album Name" Example(A Night at the Opera)
// @Param audio formData file true "Audio File (mp3/wav)"
// @Param cover formData file false "Cover Image (jpg/png)"
// @Success 201 {object} models.Track "Track successfully uploaded"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/admin/tracks/upload [post]
func (h *TrackHandler) UploadTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Limit upload size to 100MB
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		h.logger.Error("Failed to parse multipart form", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Get form fields
	title := r.FormValue("title")
	artist := r.FormValue("artist")
	album := r.FormValue("album")

	// Validate required fields
	if title == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Title is required")
		return
	}

	// Get uploaded files
	audioFile, _, err := r.FormFile("audio")
	if err != nil {
		h.logger.Error("Failed to get audio file", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Audio file is required")
		return
	}
	defer audioFile.Close()

	imageFile, _, err := r.FormFile("cover")
	if err != nil {
		// Cover image is optional
		imageFile = nil
	} else {
		defer imageFile.Close()
	}

	// Get file headers for service
	audioHeader := r.MultipartForm.File["audio"][0]
	var imageHeader *multipart.FileHeader
	if imageFile != nil {
		imageHeader = r.MultipartForm.File["cover"][0]
	}

	// Call track service
	h.logger.Info("Starting track upload",
		"user_id", userID,
		"title", title,
		"artist", artist,
		"album", album,
		"audio_file", audioHeader.Filename,
		"image_file", func() string {
			if imageHeader != nil {
				return imageHeader.Filename
			}
			return "none"
		}())

	track, err := h.trackService.UploadTrack(ctx, userID, title, artist, album, audioHeader, imageHeader)
	if err != nil {
		h.logger.Error("Failed to upload track", "error", err, "details", err.Error())
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to upload track: %v", err))
		return
	}

	sendJSONResponse(w, http.StatusCreated, track)
}

// StreamTrack handles audio streaming with Range Request support
// @Summary Stream Track Audio (Public Access)
// @Tags tracks
// @Param id path string true "Track ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {file} binary "Audio file stream"
// @Failure 404 {object} map[string]string "Not found - track does not exist"
// @Router /api/tracks/{id}/stream [get]
func (h *TrackHandler) StreamTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get track ID from URL parameter
	trackID := r.URL.Path[len("/api/tracks/"):]
	if idx := len(trackID) - len("/stream"); idx > 0 {
		trackID = trackID[:idx]
	}

	// Get track information
	track, err := h.trackService.GetTrack(ctx, trackID)
	if err != nil {
		h.logger.Error("Failed to get track", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Track not found")
		return
	}

	// Get object from MinIO through track service
	object, err := h.trackService.GetAudioFile(ctx, track.AudioFileKey)
	if err != nil {
		h.logger.Error("Failed to get object from MinIO", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get audio file")
		return
	}
	defer object.Close()

	// Get object info for size and modification time
	info, err := h.trackService.GetAudioFileInfo(ctx, track.AudioFileKey)
	if err != nil {
		h.logger.Error("Failed to get object info", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get audio info")
		return
	}

	// Create a ReadSeeker from the object
	// Note: For production with large files, you might want to implement
	// proper Range Request handling directly with MinIO SeekRead functionality
	tempFile, err := os.CreateTemp("", "stream-*.mp3")
	if err != nil {
		h.logger.Error("Failed to create temp file", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to prepare streaming")
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy object to temp file
	if _, err := io.Copy(tempFile, object); err != nil {
		h.logger.Error("Failed to copy object to temp file", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to prepare streaming")
		return
	}

	// Seek to beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		h.logger.Error("Failed to seek temp file", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to prepare streaming")
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Accept-Ranges", "bytes")

	// Use http.ServeContent to handle Range Requests properly
	modTime := info.LastModified
	if modTime.IsZero() {
		modTime = time.Now()
	}

	http.ServeContent(w, r, track.Title+".mp3", modTime, tempFile)
}

// ListTracks returns a paginated list of tracks (supports optional authentication)
// @Summary List All Tracks (Optional Auth)
// @Tags tracks
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1) Example(1)
// @Param limit query int false "Items per page" default(20) Example(20)
// @Param genre query string false "Filter by genre" Example(rock)
// @Param Authorization header string false "Bearer token for authenticated access (shows like status)" Example(Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...)
// @Success 200 {object} models.TrackListResponse "List of tracks with pagination"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/tracks [get]
func (h *TrackHandler) ListTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get user ID from context (optional)
	userID, _ := middleware.GetUserID(ctx)
	// userID will be 0 if user is not authenticated, which is fine

	// Get genre filter
	genreFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("genre")))

	// Call track service with optional user (now returns TrackResponse)
	tracks, total, err := h.trackService.ListTracksWithOptionalUser(ctx, page, limit, userID, genreFilter)
	if err != nil {
		h.logger.Error("Failed to list tracks", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to list tracks")
		return
	}

	// Prepare response with pagination metadata
	response := map[string]interface{}{
		"tracks": tracks,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	sendJSONResponse(w, http.StatusOK, response)
}

// GetTrack returns a single track by ID with optional like status for authenticated users
// @Summary Get Track by ID (Optional Auth)
// @Tags tracks
// @Accept json
// @Produce json
// @Param id path string true "Track ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Param Authorization header string false "Bearer token for authenticated access (shows like status)" Example(Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...)
// @Success 200 {object} models.Track "Track details with optional like status"
// @Failure 400 {object} map[string]string "Bad request - invalid track ID"
// @Failure 404 {object} map[string]string "Not found - track does not exist"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/tracks/{id} [get]
func (h *TrackHandler) GetTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get track ID from URL parameter
	trackID := chi.URLParam(r, "id")
	if trackID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}

	// Get user ID from context (optional)
	userID, _ := middleware.GetUserID(ctx)
	// userID will be 0 if user is not authenticated, which is fine

	// Get track with album info and optional like status
	track, err := h.trackService.GetTrackWithAlbumInfo(ctx, trackID, userID)

	if err != nil {
		h.logger.Error("Failed to get track", "track_id", trackID, "user_id", userID, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Track not found")
		return
	}

	sendJSONResponse(w, http.StatusOK, track)
}

// GetUserTracks returns all tracks for the authenticated user
// @Summary Get User's Tracks
// @Security BearerAuth
// @Tags tracks
// @Accept json
// @Produce json
// @Success 200 {object} models.UserTracksResponse "List of user's tracks"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/tracks/my [get]
func (h *TrackHandler) GetUserTracks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Call track service with album info
	tracks, err := h.trackService.GetUserTracksWithAlbumInfo(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user tracks", "user_id", userID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get user tracks")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"tracks": tracks,
	})
}

// DeleteTrack deletes a track
// @Summary Delete Track (Admin)
// @Security BearerAuth
// @Tags admin
// @Param id path string true "Track ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Success 204 "No Content - track deleted successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid track ID"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 403 {object} map[string]string "Forbidden - insufficient permissions"
// @Failure 404 {object} map[string]string "Not found - track does not exist"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/admin/tracks/{id} [delete]
func (h *TrackHandler) DeleteTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get track ID from URL parameter
	trackID := r.URL.Path[len("/api/tracks/"):]
	if trackID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}

	// Get track to verify ownership
	track, err := h.trackService.GetTrack(ctx, trackID)
	if err != nil {
		h.logger.Error("Failed to get track", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Track not found")
		return
	}

	// Check ownership
	if track.UserID != userID {
		sendErrorResponse(w, http.StatusForbidden, "You don't have permission to delete this track")
		return
	}

	// Delete track
	if err := h.trackService.DeleteTrack(ctx, trackID); err != nil {
		h.logger.Error("Failed to delete track", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to delete track")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ToggleLike toggles a like for a track
// @Summary Toggle Track Like
// @Security BearerAuth
// @Tags tracks
// @Accept json
// @Produce json
// @Param id path string true "Track ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} models.ToggleLikeResponse "Like status updated"
// @Failure 400 {object} map[string]string "Bad request - invalid track ID"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 404 {object} map[string]string "Not found - track does not exist"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/tracks/{id}/like [post]
func (h *TrackHandler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get track ID from URL parameter
	trackID := chi.URLParam(r, "id")
	if trackID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}

	// Toggle like
	isLiked, likesCount, err := h.trackService.ToggleLike(ctx, userID, trackID)
	if err != nil {
		h.logger.Error("Failed to toggle like", "track_id", trackID, "user_id", userID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to toggle like")
		return
	}

	// Return updated state
	response := map[string]interface{}{
		"liked":       isLiked,
		"likes_count": likesCount,
	}
	sendJSONResponse(w, http.StatusOK, response)
}

// IncrementPlays increments the play count for a track
// @Summary Increment Track Play Count
// @Tags tracks
// @Param id path string true "Track ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 "OK - Play count incremented"
// @Failure 400 {object} map[string]string "Bad request - invalid track ID"
// @Failure 404 {object} map[string]string "Not found - track does not exist"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/tracks/{id}/play [post]
func (h *TrackHandler) IncrementPlays(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get track ID from URL parameter
	trackID := chi.URLParam(r, "id")
	if trackID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}

	// Increment plays
	if err := h.trackService.IncrementPlays(ctx, trackID); err != nil {
		h.logger.Error("Failed to increment plays", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to increment plays")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetTrackCover serves track cover image
// @Summary Get Track Cover Image
// @Tags tracks
// @Param id path string true "Track ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {file} binary "Cover image"
// @Failure 404 {object} map[string]string "Not found - track or cover does not exist"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/tracks/{id}/cover [get]
func (h *TrackHandler) GetTrackCover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get track ID from URL parameter
	trackID := chi.URLParam(r, "id")
	if trackID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}
	
	h.logger.Info("GetTrackCover called", "track_id", trackID, "path", r.URL.Path)

	// Get track with album info
	trackResponse, err := h.trackService.GetTrackWithAlbumInfo(ctx, trackID, 0) // No user ID needed for cover
	if err != nil {
		h.logger.Error("Failed to get track with album info", "track_id", trackID, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Track not found")
		return
	}

	// Check if track has cover (from album)
	if trackResponse.CoverImageKey == "" {
		h.logger.Warn("Track has no cover image", "track_id", trackID, "cover_key", trackResponse.CoverImageKey)
		sendErrorResponse(w, http.StatusNotFound, "Track has no cover image")
		return
	}

	// Get image from MinIO through track service
	object, err := h.trackService.GetCoverImage(ctx, trackResponse.CoverImageKey)
	if err != nil {
		h.logger.Error("Failed to get cover from MinIO", "track_id", trackID, "cover_key", trackResponse.CoverImageKey, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Cover image not found")
		return
	}
	defer object.Close()

	// Get object info for content type
	info, err := h.trackService.GetCoverImageInfo(ctx, trackResponse.CoverImageKey)
	if err != nil {
		h.logger.Warn("Failed to get object info", "cover_key", trackResponse.CoverImageKey, "error", err)
	}

	// Set content type
	contentType := "image/jpeg" // default
	if info != nil && info.ContentType != "" {
		contentType = info.ContentType
	} else {
		// Try to detect from file extension
		if strings.HasSuffix(strings.ToLower(trackResponse.CoverImageKey), ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(trackResponse.CoverImageKey), ".webp") {
			contentType = "image/webp"
		}
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year

	// Copy image data to response
	if _, err := io.Copy(w, object); err != nil {
		h.logger.Error("Failed to serve cover image", "track_id", trackID, "error", err)
		return
	}
}
