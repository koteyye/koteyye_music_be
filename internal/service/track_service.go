package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"strconv"
	"strings"

	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/repository"
	"koteyye_music_be/pkg/logger"
	minioPkg "koteyye_music_be/pkg/minio"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type TrackService struct {
	trackRepo *repository.TrackRepository
	albumRepo *repository.AlbumRepository
	Minio     *minioPkg.Client
	minioSvc  *minioPkg.Service
	logger    *slog.Logger
}

func NewTrackService(trackRepo *repository.TrackRepository, albumRepo *repository.AlbumRepository, minio *minioPkg.Client, minioSvc *minioPkg.Service, log *slog.Logger) *TrackService {
	return &TrackService{
		trackRepo: trackRepo,
		albumRepo: albumRepo,
		Minio:     minio,
		minioSvc:  minioSvc,
		logger:    log,
	}
}

// UploadTrack handles the complete track upload process (DEPRECATED - use albums)
func (s *TrackService) UploadTrack(ctx context.Context, userID int, title, artist, album string, audioFile, imageFile *multipart.FileHeader) (*models.Track, error) {
	return nil, fmt.Errorf("deprecated method - use album-based track creation instead")
}

// GetTrack retrieves a track by ID
func (s *TrackService) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	// Validate and parse UUID
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid track ID format: %w", err)
	}

	track, err := s.trackRepo.GetTrackByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get track", "track_id", id, "error", err)
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	return track, nil
}

// ListTracks returns a paginated list of tracks (DEPRECATED - use ListTracksWithOptionalUser)
func (s *TrackService) ListTracks(ctx context.Context, page, limit int) ([]models.Track, int, error) {
	return nil, 0, fmt.Errorf("deprecated method - use ListTracksWithOptionalUser")
}

// ListTracksWithLikeStatus returns a paginated list of tracks with like status for the user
func (s *TrackService) ListTracksWithLikeStatus(ctx context.Context, page, limit int, userID int) ([]models.Track, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	tracks, err := s.trackRepo.ListTracksWithLikeStatus(ctx, limit, offset, userID)
	if err != nil {
		s.logger.Error("Failed to list tracks with like status", "error", err)
		return nil, 0, fmt.Errorf("failed to list tracks: %w", err)
	}

	// Get total count of tracks
	total, err := s.trackRepo.CountTracks(ctx)
	if err != nil {
		s.logger.Error("Failed to count tracks", "error", err)
		total = len(tracks) // Fallback to tracks length if count fails
	}

	return tracks, total, nil
}

// ListTracksWithOptionalUser returns a paginated list of tracks with album info and optional like status and genre filtering
// If userID is 0, returns tracks without like status for unauthenticated users
func (s *TrackService) ListTracksWithOptionalUser(ctx context.Context, page, limit int, userID int, genreFilter string) ([]models.TrackResponse, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	tracks, err := s.trackRepo.ListTracksWithAlbumInfo(ctx, limit, offset, userID, genreFilter)
	if err != nil {
		s.logger.Error("Failed to list tracks with album info", "error", err)
		return nil, 0, fmt.Errorf("failed to list tracks: %w", err)
	}

	// Generate BE endpoint URLs for all tracks
	for i := range tracks {
		// Cover URL points to track cover endpoint (which gets it from album)
		tracks[i].CoverURL = fmt.Sprintf("/tracks/%s/cover", tracks[i].ID)
		// Audio URL points to track stream endpoint
		tracks[i].AudioURL = fmt.Sprintf("/tracks/%s/stream", tracks[i].ID)
		// Add image key for frontend
		if tracks[i].CoverImageKey != "" {
			tracks[i].ImageKey = &tracks[i].CoverImageKey
		}
	}

	// Get total count of tracks
	total, err := s.trackRepo.CountTracks(ctx)
	if err != nil {
		s.logger.Error("Failed to count tracks", "error", err)
		total = len(tracks) // Fallback to tracks length if count fails
	}

	return tracks, total, nil
}

// GetUserTracks returns all tracks for a specific user (DEPRECATED - use GetUserTracksWithAlbumInfo)
func (s *TrackService) GetUserTracks(ctx context.Context, userID int) ([]models.Track, error) {
	return nil, fmt.Errorf("deprecated method - use GetUserTracksWithAlbumInfo")
}

// GetUserTracksWithAlbumInfo returns all tracks for a specific user with album info
func (s *TrackService) GetUserTracksWithAlbumInfo(ctx context.Context, userID int) ([]models.TrackResponse, error) {
	tracks, err := s.trackRepo.GetTracksByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user tracks", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user tracks: %w", err)
	}

	// Generate BE endpoint URLs for all tracks
	for i := range tracks {
		// Cover URL points to track cover endpoint (which gets it from album)
		tracks[i].CoverURL = fmt.Sprintf("/tracks/%s/cover", tracks[i].ID)
		// Audio URL points to track stream endpoint
		tracks[i].AudioURL = fmt.Sprintf("/tracks/%s/stream", tracks[i].ID)
		// Add image key for frontend
		if tracks[i].CoverImageKey != "" {
			tracks[i].ImageKey = &tracks[i].CoverImageKey
		}
	}

	return tracks, nil
}

// DeleteTrack deletes a track by ID
func (s *TrackService) DeleteTrack(ctx context.Context, id string) error {
	// Validate and parse UUID
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid track ID format: %w", err)
	}

	// Get track info first to delete from S3
	track, err := s.trackRepo.GetTrackByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get track for deletion", "track_id", id, "error", err)
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Delete audio from MinIO
	if err := s.minioSvc.DeleteFile(ctx, "music-files", track.AudioFileKey); err != nil {
		s.logger.Error("Failed to delete audio from MinIO", "track_id", id, "error", err)
		// Continue even if MinIO deletion fails
	}

	// Note: In new album architecture, cover images belong to albums, not tracks

	// Delete from database
	if err := s.trackRepo.DeleteTrack(ctx, id); err != nil {
		s.logger.Error("Failed to delete track from database", "track_id", id, "error", err)
		return fmt.Errorf("failed to delete track: %w", err)
	}

	s.logger.Info("Track deleted successfully", "track_id", id)

	return nil
}

// saveUploadedFile saves a multipart file to a local path
func (s *TrackService) saveUploadedFile(file *multipart.FileHeader, path string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

// convertAudioToMP3 converts audio file to MP3 using ffmpeg and returns duration in seconds
func (s *TrackService) convertAudioToMP3(inputPath, outputPath string) (float64, error) {
	s.logger.Info("Starting audio conversion", "input", inputPath, "output", outputPath)

	// First, get duration
	duration, err := s.getAudioDuration(inputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get audio duration: %w", err)
	}

	s.logger.Info("Audio duration detected", "duration", duration)

	// Convert to MP3 at 320kbps
	args := []string{
		"-i", inputPath,
		"-codec:a", "libmp3lame",
		"-b:a", "320k",
		"-y", // Overwrite output file if exists
		outputPath,
	}

	s.logger.Info("Running ffmpeg conversion", "args", args)
	if err := s.runFFmpeg(args); err != nil {
		return 0, err
	}

	s.logger.Info("Audio conversion completed successfully")
	return duration, nil
}

// getAudioDuration extracts duration from audio file using ffprobe
func (s *TrackService) getAudioDuration(filePath string) (float64, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath,
	}

	output, err := s.runFFprobe(args)
	if err != nil {
		return 0, err
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// runFFmpeg executes ffmpeg command
func (s *TrackService) runFFmpeg(args []string) error {
	cmd := logger.NewCommand("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "command not found") {
			return fmt.Errorf("ffmpeg is not installed. Please install ffmpeg to enable audio processing: %w", err)
		}
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	return nil
}

// runFFprobe executes ffprobe command
func (s *TrackService) runFFprobe(args []string) (string, error) {
	cmd := logger.NewCommand("ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "command not found") {
			return "", fmt.Errorf("ffprobe is not installed. Please install ffmpeg to enable audio processing: %w", err)
		}
		return "", fmt.Errorf("ffprobe execution failed: %w", err)
	}

	return string(output), nil
}

// getImageContentType returns the appropriate content type for an image file
func (s *TrackService) getImageContentType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// ToggleLike toggles a like for a track
// Returns (isLiked, newLikesCount, error)
func (s *TrackService) ToggleLike(ctx context.Context, userID int, trackID string) (bool, int, error) {
	// Validate and parse UUID
	if _, err := uuid.Parse(trackID); err != nil {
		return false, 0, fmt.Errorf("invalid track ID format: %w", err)
	}

	isLiked, likesCount, err := s.trackRepo.ToggleLike(ctx, userID, trackID)
	if err != nil {
		s.logger.Error("Failed to toggle like", "user_id", userID, "track_id", trackID, "error", err)
		return false, 0, fmt.Errorf("failed to toggle like: %w", err)
	}

	s.logger.Info("Like toggled", "user_id", userID, "track_id", trackID, "liked", isLiked)

	return isLiked, likesCount, nil
}

// IncrementPlays increments the play count for a track
func (s *TrackService) IncrementPlays(ctx context.Context, trackID string) error {
	// Validate and parse UUID
	if _, err := uuid.Parse(trackID); err != nil {
		return fmt.Errorf("invalid track ID format: %w", err)
	}

	if err := s.trackRepo.IncrementPlays(ctx, trackID); err != nil {
		s.logger.Error("Failed to increment plays", "track_id", trackID, "error", err)
		return fmt.Errorf("failed to increment plays: %w", err)
	}

	return nil
}

// GetTrackWithAlbumInfo retrieves a track by ID with album info and like status
func (s *TrackService) GetTrackWithAlbumInfo(ctx context.Context, trackID string, userID int) (*models.TrackResponse, error) {
	// Validate and parse UUID
	if _, err := uuid.Parse(trackID); err != nil {
		return nil, fmt.Errorf("invalid track ID format: %w", err)
	}

	track, err := s.trackRepo.GetTrackWithAlbumInfo(ctx, trackID, userID)
	if err != nil {
		s.logger.Error("Failed to get track with album info", "track_id", trackID, "error", err)
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	// Generate BE endpoint URL for cover
	track.CoverURL = fmt.Sprintf("/tracks/%s/cover", track.ID)

	return track, nil
}

// GetUserLikedTrackIDs returns a list of track IDs liked by the user
func (s *TrackService) GetUserLikedTrackIDs(ctx context.Context, userID int) ([]string, error) {
	trackIDs, err := s.trackRepo.GetUserLikedTrackIDs(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user liked track IDs", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user liked track IDs: %w", err)
	}

	return trackIDs, nil
}

// GetTrackStats returns play and like counts for a track
func (s *TrackService) GetTrackStats(ctx context.Context, trackID string) (*models.TrackStats, error) {
	// Validate and parse UUID
	if _, err := uuid.Parse(trackID); err != nil {
		return nil, fmt.Errorf("invalid track ID format: %w", err)
	}

	stats, err := s.trackRepo.GetTrackStats(ctx, trackID)
	if err != nil {
		s.logger.Error("Failed to get track stats", "track_id", trackID, "error", err)
		return nil, fmt.Errorf("failed to get track stats: %w", err)
	}

	return stats, nil
}

// GetCoverImage returns the cover image object from MinIO for a track
func (s *TrackService) GetCoverImage(ctx context.Context, coverKey string) (io.ReadCloser, error) {
	return s.minioSvc.GetObject(ctx, coverKey)
}

// GetCoverImageInfo returns the cover image info from MinIO for a track  
func (s *TrackService) GetCoverImageInfo(ctx context.Context, coverKey string) (*minio.ObjectInfo, error) {
	return s.minioSvc.GetObjectInfo(ctx, coverKey)
}

// GetAudioFile returns the audio file object from MinIO
func (s *TrackService) GetAudioFile(ctx context.Context, audioKey string) (io.ReadCloser, error) {
	return s.minioSvc.GetObject(ctx, audioKey)
}

// GetAudioFileInfo returns the audio file info from MinIO  
func (s *TrackService) GetAudioFileInfo(ctx context.Context, audioKey string) (*minio.ObjectInfo, error) {
	return s.minioSvc.GetObjectInfo(ctx, audioKey)
}
