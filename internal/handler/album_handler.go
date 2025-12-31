package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"koteyye_music_be/internal/service"
)

type AlbumHandler struct {
	albumService *service.AlbumService
	logger       *slog.Logger
}

func NewAlbumHandler(albumService *service.AlbumService, log *slog.Logger) *AlbumHandler {
	return &AlbumHandler{
		albumService: albumService,
		logger:       log,
	}
}

// GetAlbums returns a list of all albums with optional genre filtering
// @Summary Get Albums
// @Tags albums
// @Produce json
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Param genre query string false "Filter by genre" example(rock)
// @Success 200 {array} models.AlbumResponse
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/albums [get]
func (h *AlbumHandler) GetAlbums(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Get genre filter
	genreFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("genre")))

	// Get albums
	albums, err := h.albumService.GetAllAlbums(ctx, limit, offset, genreFilter)
	if err != nil {
		h.logger.Error("Failed to get albums", "error", err)
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get albums")
		return
	}

	h.logger.Info("Albums retrieved successfully", "count", len(albums), "page", page)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(albums)
}

// GetAlbumByID returns album details with tracks
// @Summary Get Album Details
// @Tags albums
// @Produce json
// @Param id path string true "Album ID"
// @Success 200 {object} models.AlbumDetail
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Album not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/albums/{id} [get]
func (h *AlbumHandler) GetAlbumByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get album ID from URL
	albumID := strings.TrimPrefix(r.URL.Path, "/api/albums/")
	if albumID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Album ID is required")
		return
	}

	// Get album with tracks
	albumDetail, err := h.albumService.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		h.logger.Error("Failed to get album", "album_id", albumID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			sendErrorResponse(w, http.StatusNotFound, "Album not found")
			return
		}
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get album")
		return
	}

	h.logger.Info("Album retrieved successfully", "album_id", albumID, "tracks_count", len(albumDetail.Tracks))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(albumDetail)
}

// GetAlbumInfo returns basic album information without tracks
// @Summary Get Album Info
// @Tags albums
// @Produce json
// @Param id path string true "Album ID"
// @Success 200 {object} models.AlbumResponse
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Album not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/albums/{id}/info [get]
func (h *AlbumHandler) GetAlbumInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get album ID from URL
	albumID := strings.TrimPrefix(r.URL.Path, "/api/albums/")
	albumID = strings.TrimSuffix(albumID, "/info")
	if albumID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Album ID is required")
		return
	}

	// Get album info
	album, err := h.albumService.GetAlbumByID(ctx, albumID)
	if err != nil {
		h.logger.Error("Failed to get album info", "album_id", albumID, "error", err)
		if strings.Contains(err.Error(), "not found") {
			sendErrorResponse(w, http.StatusNotFound, "Album not found")
			return
		}
		sendErrorResponse(w, http.StatusInternalServerError, "Failed to get album info")
		return
	}

	h.logger.Info("Album info retrieved successfully", "album_id", albumID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)
}

// GetAlbumCover returns the cover image for an album
// @Summary Get Album Cover Image
// @Tags albums
// @Param id path string true "Album ID" Example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {file} binary "Cover image"
// @Failure 404 {object} map[string]string "Not found - album or cover does not exist"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/albums/{id}/cover [get]
func (h *AlbumHandler) GetAlbumCover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get album ID from URL parameter
	albumID := chi.URLParam(r, "id")
	if albumID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Album ID is required")
		return
	}

	// Get album info
	album, err := h.albumService.GetAlbumRaw(ctx, albumID)
	if err != nil {
		h.logger.Error("Failed to get album", "album_id", albumID, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Album not found")
		return
	}

	// Check if album has cover
	if album.CoverImageKey == "" {
		sendErrorResponse(w, http.StatusNotFound, "Album has no cover image")
		return
	}

	// Get image from MinIO through album service
	object, err := h.albumService.GetCoverImage(ctx, album.CoverImageKey)
	if err != nil {
		h.logger.Error("Failed to get cover from MinIO", "album_id", albumID, "cover_key", album.CoverImageKey, "error", err)
		sendErrorResponse(w, http.StatusNotFound, "Cover image not found")
		return
	}
	defer object.Close()

	// Get object info for content type
	info, err := h.albumService.GetCoverImageInfo(ctx, album.CoverImageKey)
	if err != nil {
		h.logger.Warn("Failed to get object info", "cover_key", album.CoverImageKey, "error", err)
	}

	// Set content type
	contentType := "image/jpeg" // default
	if info != nil && info.ContentType != "" {
		contentType = info.ContentType
	} else {
		// Try to detect from file extension
		if strings.HasSuffix(strings.ToLower(album.CoverImageKey), ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(album.CoverImageKey), ".webp") {
			contentType = "image/webp"
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year

	// Stream the file to the response
	_, err = io.Copy(w, object)
	if err != nil {
		h.logger.Error("Failed to stream cover image", "album_id", albumID, "error", err)
		// Can't send error response here as we already started writing the body
		return
	}

	h.logger.Info("Album cover served successfully", "album_id", albumID)
}
