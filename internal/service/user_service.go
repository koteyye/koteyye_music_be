package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strings"

	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/repository"
	"koteyye_music_be/pkg/minio"

	"github.com/google/uuid"
)

type UserService struct {
	userRepo    *repository.UserRepository
	minioClient *minio.Client
	logger      *slog.Logger
}

func NewUserService(userRepo *repository.UserRepository, minioClient *minio.Client, log *slog.Logger) *UserService {
	return &UserService{
		userRepo:    userRepo,
		minioClient: minioClient,
		logger:      log,
	}
}

// GetUserProfile retrieves user profile by ID with last track details
func (s *UserService) GetUserProfile(ctx context.Context, userID int) (*models.UserProfileResponse, error) {
	user, err := s.userRepo.GetUserWithLastTrack(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user with last track", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Convert avatar key to URL
	var avatarURL *string
	if user.AvatarKey != nil && *user.AvatarKey != "" {
		url := fmt.Sprintf("/api/avatars/%s", *user.AvatarKey)
		avatarURL = &url
	}

	profile := &models.UserProfileResponse{
		ID:               user.ID,
		Email:            user.Email,
		Name:             user.Name,
		AvatarURL:        avatarURL,
		Provider:         user.Provider,
		Role:             user.Role,
		LastTrackID:      user.LastTrackID,
		LastPosition:     user.LastPosition,
		VolumePreference: user.VolumePreference,
		LastTrack:        user.LastTrack,
		LastLoginAt:      user.LastLoginAt,
		CreatedAt:        user.CreatedAt,
	}

	return profile, nil
}

// UpdateUserProfile updates user profile information
func (s *UserService) UpdateUserProfile(ctx context.Context, userID int, req *models.UpdateProfileRequest) (*models.UserProfileResponse, error) {
	// Update profile in database
	if err := s.userRepo.UpdateUserProfile(ctx, userID, req.Name, req.AvatarKey); err != nil {
		s.logger.Error("Failed to update user profile", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	// Return updated profile
	profile, err := s.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.logger.Info("User profile updated successfully", "user_id", userID)
	return profile, nil
}

// UploadAvatar uploads user avatar to MinIO and updates profile
func (s *UserService) UploadAvatar(ctx context.Context, userID int, file multipart.File, header *multipart.FileHeader) (*models.UserProfileResponse, error) {
	// Validate file type
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedTypes[ext] {
		return nil, fmt.Errorf("unsupported file type: %s. Allowed: jpg, jpeg, png, gif, webp", ext)
	}

	// Validate file size (max 5MB)
	if header.Size > 5*1024*1024 {
		return nil, fmt.Errorf("file too large: %d bytes. Maximum allowed: 5MB", header.Size)
	}

	// Generate unique filename
	avatarKey := fmt.Sprintf("avatars/%d/%s%s", userID, uuid.New().String(), ext)

	// Upload to MinIO
	_, err := s.minioClient.PutObject(ctx, avatarKey, file, header.Size, map[string]string{
		"Content-Type": header.Header.Get("Content-Type"),
	})
	if err != nil {
		s.logger.Error("Failed to upload avatar to MinIO", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to upload avatar: %w", err)
	}

	// Get current user to check for old avatar
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get user for old avatar cleanup", "user_id", userID, "error", err)
		// Continue anyway, don't fail the operation
	} else if user.AvatarKey != nil && *user.AvatarKey != "" {
		// Delete old avatar if exists
		if strings.HasPrefix(*user.AvatarKey, "avatars/") {
			if err := s.minioClient.DeleteObject(ctx, *user.AvatarKey); err != nil {
				s.logger.Warn("Failed to remove old avatar", "user_id", userID, "old_key", *user.AvatarKey, "error", err)
				// Continue anyway
			}
		}
	}

	// Update user profile with new avatar key
	if err := s.userRepo.UpdateUserProfile(ctx, userID, nil, &avatarKey); err != nil {
		// Try to cleanup uploaded file
		s.minioClient.DeleteObject(ctx, avatarKey)
		s.logger.Error("Failed to update user avatar key", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to update profile with new avatar: %w", err)
	}

	s.logger.Info("Avatar uploaded successfully", "user_id", userID, "avatar_key", avatarKey)

	// Return updated profile
	return s.GetUserProfile(ctx, userID)
}

// RemoveAvatar removes user avatar from MinIO and updates profile
func (s *UserService) RemoveAvatar(ctx context.Context, userID int) (*models.UserProfileResponse, error) {
	// Get current user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Remove avatar from MinIO if exists and it's stored in our MinIO
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		if strings.Contains(*user.AvatarURL, "avatars/") {
			avatarKey := extractMinIOKeyFromURL(*user.AvatarURL)
			if avatarKey != "" {
				if err := s.minioClient.DeleteObject(ctx, avatarKey); err != nil {
					s.logger.Warn("Failed to remove avatar from MinIO", "user_id", userID, "avatar_key", avatarKey, "error", err)
					// Continue anyway, still update database
				}
			}
		}
	}

	// Update user profile to remove avatar URL
	if err := s.userRepo.UpdateUserProfile(ctx, userID, nil, nil); err != nil {
		s.logger.Error("Failed to update user profile", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to remove avatar from profile: %w", err)
	}

	s.logger.Info("Avatar removed successfully", "user_id", userID)

	// Return updated profile
	return s.GetUserProfile(ctx, userID)
}

// GetAvatarStream returns avatar file stream from MinIO
func (s *UserService) GetAvatarStream(ctx context.Context, avatarKey string) (io.ReadCloser, string, int64, error) {
	// Validate that the key is for avatars
	if !strings.HasPrefix(avatarKey, "avatars/") {
		return nil, "", 0, fmt.Errorf("invalid avatar key")
	}

	// Get object from MinIO
	object, err := s.minioClient.GetObject(ctx, avatarKey)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get avatar: %w", err)
	}

	// Get object info for content type and size
	info, err := s.minioClient.GetObjectInfo(ctx, avatarKey)
	if err != nil {
		object.Close()
		return nil, "", 0, fmt.Errorf("failed to get avatar info: %w", err)
	}

	contentType := info.Metadata.Get("Content-Type")
	if contentType == "" {
		// Determine content type from file extension
		ext := strings.ToLower(filepath.Ext(avatarKey))
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		default:
			contentType = "application/octet-stream"
		}
	}

	return object, contentType, info.Size, nil
}

// UpdatePlayerState updates user's player state (track, position, volume)
func (s *UserService) UpdatePlayerState(ctx context.Context, userID int, trackID string, position float64, volume int) error {
	// Validate that track exists (this prevents foreign key constraint violations)
	exists, err := s.userRepo.TrackExists(ctx, trackID)
	if err != nil {
		s.logger.Error("Failed to check if track exists", "track_id", trackID, "error", err)
		return fmt.Errorf("failed to validate track: %w", err)
	}
	if !exists {
		s.logger.Warn("Track not found for player state update", "track_id", trackID)
		return fmt.Errorf("track not found")
	}

	err = s.userRepo.UpdatePlayerState(ctx, userID, trackID, position, volume)
	if err != nil {
		s.logger.Error("Failed to update player state", "user_id", userID, "error", err)
		return fmt.Errorf("failed to update player state: %w", err)
	}

	s.logger.Debug("Player state updated successfully", "user_id", userID, "track_id", trackID, "position", position, "volume", volume)
	return nil
}

// GetUserWithLastTrack retrieves user with full last track details
func (s *UserService) GetUserWithLastTrack(ctx context.Context, userID int) (*models.User, error) {
	user, err := s.userRepo.GetUserWithLastTrack(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user with last track", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get user with last track: %w", err)
	}

	return user, nil
}

// extractMinIOKeyFromURL extracts MinIO object key from avatar URL
func extractMinIOKeyFromURL(url string) string {
	// Extract key from URL like "/api/avatars/avatars/123/uuid.jpg"
	if strings.HasPrefix(url, "/api/avatars/") {
		return url[len("/api/avatars/"):]
	}
	return ""
}
