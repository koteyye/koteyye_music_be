package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

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
