package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"koteyye_music_be/internal/middleware"
	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/service"
)

type AdminHandler struct {
	trackService *service.TrackService
	albumService *service.AlbumService
	logger       *slog.Logger
}

func NewAdminHandler(trackService *service.TrackService, albumService *service.AlbumService, log *slog.Logger) *AdminHandler {
	return &AdminHandler{
		trackService: trackService,
		albumService: albumService,
		logger:       log,
	}
}

// CreateAlbum creates a new album with cover image (admin only)
// @Summary Create Album
// @Security BearerAuth
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "Album title"
// @Param artist formData string true "Artist name"
// @Param genre formData string true "Music genre (pop, rock, hip-hop, rap, indie, electronic, house, techno, jazz, blues, classical, metal, punk, r-n-b, soul, folk, reggae, country, latin, k-pop, soundtrack, lo-fi, chanson)"
// @Param release_date formData string true "Release date (YYYY-MM-DD)"
// @Param cover formData file true "Album cover image (JPG, PNG)"
// @Success 201 {object} models.AlbumResponse
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/admin/albums [post]
func (h *AdminHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.logger.Error("Failed to parse multipart form", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Get form fields
	title := r.FormValue("title")
	artist := r.FormValue("artist")
	genre := r.FormValue("genre")
	releaseDate := r.FormValue("release_date")

	if title == "" || artist == "" || genre == "" || releaseDate == "" {
		sendErrorResponse(w, http.StatusBadRequest, "All fields (title, artist, genre, release_date) are required")
		return
	}

	// Get cover file
	coverFile, coverHeader, err := r.FormFile("cover")
	if err != nil {
		h.logger.Error("Failed to get cover file", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Cover image is required")
		return
	}
	defer coverFile.Close()

	// Create album request
	albumReq := &models.AlbumCreate{
		Title:       title,
		Artist:      artist,
		Genre:       genre,
		ReleaseDate: releaseDate,
	}

	// Create album
	album, err := h.albumService.CreateAlbum(ctx, albumReq, coverFile, coverHeader)
	if err != nil {
		h.logger.Error("Failed to create album", "error", err)
		if strings.Contains(err.Error(), "invalid genre") {
			sendErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to create album")
		return
	}

	h.logger.Info("Album created successfully", "album_id", album.ID, "title", album.Title)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(album)
}

// AddTrackToAlbum adds a track to an existing album (admin only)
// @Summary Add Track to Album
// @Security BearerAuth
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Album ID"
// @Param title formData string true "Track title"
// @Param artist formData string false "Track artist (optional, uses album artist if empty)"
// @Param audio formData file true "Audio file (MP3, WAV, M4A, FLAC)"
// @Success 201 {object} models.TrackResponse
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Album not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/admin/albums/{id}/tracks [post]
func (h *AdminHandler) AddTrackToAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get album ID from URL
	albumID := strings.TrimPrefix(r.URL.Path, "/api/admin/albums/")
	albumID = strings.TrimSuffix(albumID, "/tracks")
	if albumID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Album ID is required")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.logger.Error("Failed to parse multipart form", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	// Get form fields
	title := r.FormValue("title")
	artist := r.FormValue("artist")

	if title == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Title is required")
		return
	}

	// Get audio file
	audioFile, audioHeader, err := r.FormFile("audio")
	if err != nil {
		h.logger.Error("Failed to get audio file", "error", err)
		sendErrorResponse(w, http.StatusBadRequest, "Audio file is required")
		return
	}
	defer audioFile.Close()

	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "User not found")
		return
	}

	// Create track request
	var artistPtr *string
	if artist != "" {
		artistPtr = &artist
	}

	trackReq := &models.TrackCreate{
		AlbumID: albumID,
		Title:   title,
		Artist:  artistPtr,
	}

	// Add track to album
	track, err := h.albumService.AddTrackToAlbum(ctx, albumID, userID, trackReq, audioFile, audioHeader)
	if err != nil {
		h.logger.Error("Failed to add track to album", "album_id", albumID, "error", err)
		if strings.Contains(err.Error(), "album not found") {
			sendErrorResponse(w, http.StatusNotFound, "Album not found")
			return
		}
		if strings.Contains(err.Error(), "invalid audio format") {
			sendErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to add track to album")
		return
	}

	h.logger.Info("Track added to album successfully", "album_id", albumID, "track_id", track.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(track)
}

// DeleteAlbum deletes an album and all its tracks (admin only)
// @Summary Delete Album
// @Security BearerAuth
// @Tags admin
// @Param id path string true "Album ID"
// @Success 204 "No Content - album deleted successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Album not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/admin/albums/{id} [delete]
func (h *AdminHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get album ID from URL
	albumID := strings.TrimPrefix(r.URL.Path, "/api/admin/albums/")
	if albumID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Album ID is required")
		return
	}

	h.logger.Info("Admin deleting album", "album_id", albumID)

	// Delete album
	if err := h.albumService.DeleteAlbum(ctx, albumID); err != nil {
		h.logger.Error("Failed to delete album", "album_id", albumID, "error", err)
		if strings.Contains(err.Error(), "album not found") {
			sendErrorResponse(w, http.StatusNotFound, "Album not found")
			return
		}
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to delete album")
		return
	}

	h.logger.Info("Album deleted successfully by admin", "album_id", albumID)
	w.WriteHeader(http.StatusNoContent)
}

// DeleteTrack deletes a track by ID (admin only)
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
func (h *AdminHandler) DeleteTrack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get track ID from URL parameter
	trackID := r.URL.Path[len("/api/admin/tracks/"):]
	if trackID == "" {
		h.logger.Warn("Delete track request without ID")
		sendErrorResponse(w, http.StatusBadRequest, "Track ID is required")
		return
	}

	h.logger.Info("Admin deleting track", "track_id", trackID)

	// Delete track (service handles DB and MinIO deletion with consistency)
	if err := h.trackService.DeleteTrack(ctx, trackID); err != nil {
		h.logger.Error("Failed to delete track", "track_id", trackID, "error", err)
		if err.Error() == "track not found" {
			sendErrorResponse(w, http.StatusNotFound, "Track not found")
			return
		}
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to delete track")
		return
	}

	h.logger.Info("Track deleted successfully by admin", "track_id", trackID)
	w.WriteHeader(http.StatusNoContent)
}
