package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/repository"
	"koteyye_music_be/pkg/audio"
	minioPkg "koteyye_music_be/pkg/minio"
)

type AlbumService struct {
	albumRepo *repository.AlbumRepository
	trackRepo *repository.TrackRepository
	minioSvc  *minioPkg.Service
}

func NewAlbumService(albumRepo *repository.AlbumRepository, trackRepo *repository.TrackRepository, minioSvc *minioPkg.Service) *AlbumService {
	return &AlbumService{
		albumRepo: albumRepo,
		trackRepo: trackRepo,
		minioSvc:  minioSvc,
	}
}

func (s *AlbumService) CreateAlbum(ctx context.Context, req *models.AlbumCreate, coverFile multipart.File, coverHeader *multipart.FileHeader) (*models.AlbumResponse, error) {
	// Validate genre
	if !models.IsValidGenre(req.Genre) {
		return nil, fmt.Errorf("invalid genre: %s. Allowed genres: %v", req.Genre, models.AllowedGenres)
	}

	// Normalize genre to lowercase
	normalizedGenre := strings.ToLower(req.Genre)

	// Validate file type
	if !isValidImageFile(coverHeader.Filename) {
		return nil, fmt.Errorf("invalid cover image format. Allowed: jpg, jpeg, png")
	}

	// Generate album ID and cover path
	albumID := uuid.New().String()
	coverExt := filepath.Ext(coverHeader.Filename)
	if coverExt == "" {
		coverExt = ".jpg"
	}
	coverKey := fmt.Sprintf("albums/%s/cover%s", albumID, coverExt)

	// Upload cover to MinIO
	_, err := s.minioSvc.UploadFile(ctx, "music-files", coverKey, coverFile, coverHeader.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to upload cover image: %w", err)
	}

	// Parse release date
	releaseDate, err := time.Parse("2006-01-02", req.ReleaseDate)
	if err != nil {
		return nil, fmt.Errorf("invalid release date format. Use YYYY-MM-DD: %w", err)
	}

	// Create album record
	album := &models.Album{
		ID:            albumID,
		Title:         req.Title,
		Artist:        req.Artist,
		ReleaseDate:   releaseDate,
		Genre:         normalizedGenre,
		CoverImageKey: coverKey,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = s.albumRepo.Create(ctx, album)
	if err != nil {
		// Cleanup uploaded cover on database error
		s.minioSvc.DeleteFile(ctx, "music-files", coverKey)
		return nil, fmt.Errorf("failed to create album: %w", err)
	}

	// Generate BE endpoint URL for cover
	coverURL := fmt.Sprintf("/albums/%s/cover", albumID)
	year := releaseDate.Year()

	return &models.AlbumResponse{
		ID:          albumID,
		Title:       req.Title,
		Artist:      req.Artist,
		ReleaseDate: releaseDate.Format("2006-01-02"),
		Genre:       normalizedGenre,
		CoverURL:    coverURL,
		Year:        year,
		CreatedAt:   album.CreatedAt,
	}, nil
}

func (s *AlbumService) GetAlbumByID(ctx context.Context, id string) (*models.AlbumResponse, error) {
	album, err := s.albumRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("album not found: %w", err)
	}

	// Generate BE endpoint URL for cover
	coverURL := fmt.Sprintf("/albums/%s/cover", album.ID)

	// Format release date and extract year
	releaseDateStr := album.ReleaseDate.Format("2006-01-02")
	year := album.ReleaseDate.Year()

	return &models.AlbumResponse{
		ID:          album.ID,
		Title:       album.Title,
		Artist:      album.Artist,
		ReleaseDate: releaseDateStr,
		Genre:       album.Genre,
		CoverURL:    coverURL,
		Year:        year,
		CreatedAt:   album.CreatedAt,
	}, nil
}

// GetAlbumRaw returns raw album data with internal fields for handlers
func (s *AlbumService) GetAlbumRaw(ctx context.Context, id string) (*models.Album, error) {
	return s.albumRepo.GetByID(ctx, id)
}

func (s *AlbumService) GetAllAlbums(ctx context.Context, limit, offset int, genreFilter string) ([]models.AlbumResponse, error) {
	albums, err := s.albumRepo.GetAll(ctx, limit, offset, genreFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get albums: %w", err)
	}

	var responses []models.AlbumResponse
	for _, album := range albums {
		coverURL := fmt.Sprintf("/api/albums/%s/cover", album.ID)
		releaseDateStr := album.ReleaseDate.Format("2006-01-02")
		year := album.ReleaseDate.Year()

		responses = append(responses, models.AlbumResponse{
			ID:          album.ID,
			Title:       album.Title,
			Artist:      album.Artist,
			ReleaseDate: releaseDateStr,
			Genre:       album.Genre,
			CoverURL:    coverURL,
			Year:        year,
			CreatedAt:   album.CreatedAt,
		})
	}

	return responses, nil
}

func (s *AlbumService) GetAlbumWithTracks(ctx context.Context, albumID string) (*models.AlbumDetail, error) {
	albumDetail, err := s.albumRepo.GetAlbumWithTracks(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album with tracks: %w", err)
	}

	// Generate BE endpoint URL for album cover
	albumDetail.Album.CoverURL = fmt.Sprintf("/albums/%s/cover", albumID)

	// Generate BE endpoint URLs for tracks
	for i := range albumDetail.Tracks {
		// Cover URL points to track cover endpoint (which gets it from album)
		albumDetail.Tracks[i].CoverURL = fmt.Sprintf("/tracks/%s/cover", albumDetail.Tracks[i].ID)
		// Audio URL points to track stream endpoint
		albumDetail.Tracks[i].AudioURL = fmt.Sprintf("/tracks/%s/stream", albumDetail.Tracks[i].ID)
	}

	return albumDetail, nil
}

func (s *AlbumService) DeleteAlbum(ctx context.Context, albumID string) error {
	// Verify album exists before deletion
	_, err := s.albumRepo.GetByID(ctx, albumID)
	if err != nil {
		return fmt.Errorf("album not found: %w", err)
	}

	// Delete album from database (this will cascade delete tracks)
	err = s.albumRepo.Delete(ctx, albumID)
	if err != nil {
		return fmt.Errorf("failed to delete album: %w", err)
	}

	// Delete album folder from MinIO (includes cover and all tracks)
	folderPath := fmt.Sprintf("albums/%s/", albumID)
	err = s.minioSvc.DeleteFolder(ctx, "music-files", folderPath)
	if err != nil {
		// Log error but don't fail - database deletion already succeeded
		// In production, this should be handled by a cleanup job
		fmt.Printf("Warning: failed to delete album folder from storage: %v\n", err)
	}

	return nil
}

// GetCoverImage returns the cover image object from MinIO
func (s *AlbumService) GetCoverImage(ctx context.Context, coverKey string) (io.ReadCloser, error) {
	return s.minioSvc.GetObject(ctx, coverKey)
}

// GetCoverImageInfo returns the cover image info from MinIO
func (s *AlbumService) GetCoverImageInfo(ctx context.Context, coverKey string) (*minio.ObjectInfo, error) {
	return s.minioSvc.GetObjectInfo(ctx, coverKey)
}

func (s *AlbumService) AddTrackToAlbum(ctx context.Context, albumID string, userID int, req *models.TrackCreate, audioFile multipart.File, audioHeader *multipart.FileHeader) (*models.TrackResponse, error) {
	// Verify album exists
	album, err := s.albumRepo.GetByID(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("album not found: %w", err)
	}

	// Validate audio file
	if !isValidAudioFile(audioHeader.Filename) {
		return nil, fmt.Errorf("invalid audio format. Allowed: mp3, wav, m4a, flac")
	}

	// Extract metadata from audio file (duration, format, etc.)
	metadata, err := audio.ExtractMetadata(audioFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio metadata: %w", err)
	}

	// Validate that it's actually an audio file
	if !metadata.IsValidAudioFormat() {
		return nil, fmt.Errorf("invalid audio format detected: %s", metadata.Format)
	}

	// Generate track ID and audio path
	trackID := uuid.New().String()
	audioKey := fmt.Sprintf("albums/%s/%s.mp3", albumID, trackID)

	// Upload audio file to MinIO
	_, err = s.minioSvc.UploadFile(ctx, "music-files", audioKey, audioFile, audioHeader.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio file: %w", err)
	}

	// Create track record
	track := &models.Track{
		ID:              trackID,
		UserID:          userID,
		AlbumID:         albumID,
		Title:           req.Title,
		Artist:          req.Artist,
		DurationSeconds: metadata.GetDurationSeconds(),
		AudioFileKey:    audioKey,
		PlaysCount:      0,
		LikesCount:      0,
		CreatedAt:       time.Now(),
	}

	err = s.trackRepo.CreateTrack(ctx, track)
	if err != nil {
		// Cleanup uploaded audio on database error
		s.minioSvc.DeleteFile(ctx, "music-files", audioKey)
		return nil, fmt.Errorf("failed to create track: %w", err)
	}

	// Generate BE endpoint URLs
	coverURL := fmt.Sprintf("/api/tracks/%s/cover", trackID)
	audioURL := fmt.Sprintf("/api/tracks/%s/stream", trackID)

	// Determine final artist name
	finalArtist := album.Artist
	if req.Artist != nil && *req.Artist != "" {
		finalArtist = *req.Artist
	}

	return &models.TrackResponse{
		ID:              trackID,
		Title:           req.Title,
		ArtistName:      finalArtist,
		AlbumID:         albumID,
		AlbumTitle:      album.Title,
		CoverURL:        coverURL,
		AudioURL:        audioURL,
		AudioFileKey:    audioKey,
		ReleaseDate:     album.ReleaseDate.Format("2006-01-02"),
		Genre:           album.Genre,
		DurationSeconds: metadata.GetDurationSeconds(),
		PlaysCount:      0,
		LikesCount:      0,
		IsLiked:         false,
		CreatedAt:       track.CreatedAt,
	}, nil
}

func isValidImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func isValidAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".mp3", ".wav", ".m4a", ".flac"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}
