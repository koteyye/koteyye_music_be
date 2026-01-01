package repository

import (
	"context"
	"fmt"
	"time"

	"koteyye_music_be/internal/models"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, name, avatar_key, password_hash, provider, external_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, name, avatar_key, password_hash, provider, external_id, role, 
		          last_track_id, last_position, volume_preference, created_at, last_login_at
	`

	// Use NULL for nil pointers (guest users)
	var email, name, avatarKey, passwordHash, provider, externalID interface{} = user.Email, user.Name, user.AvatarKey, user.PasswordHash, user.Provider, user.ExternalID

	err := r.db.Pool.QueryRow(ctx, query, email, name, avatarKey, passwordHash, provider, externalID, user.Role).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AvatarKey,
		&user.PasswordHash,
		&user.Provider,
		&user.ExternalID,
		&user.Role,
		&user.LastTrackID,
		&user.LastPosition,
		&user.VolumePreference,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, name, avatar_key, password_hash, provider, external_id, role, created_at, last_login_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AvatarKey,
		&user.PasswordHash,
		&user.Provider,
		&user.ExternalID,
		&user.Role,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, email, name, avatar_key, password_hash, provider, external_id, role, 
		       last_track_id, last_position, volume_preference, created_at, last_login_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AvatarKey,
		&user.PasswordHash,
		&user.Provider,
		&user.ExternalID,
		&user.Role,
		&user.LastTrackID,
		&user.LastPosition,
		&user.VolumePreference,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// CreateOAuthUser creates a new user via OAuth
func (r *UserRepository) CreateOAuthUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, provider, external_id)
		VALUES ($1, NULL, $2, $3)
		RETURNING id, created_at, role
	`

	err := r.db.Pool.QueryRow(ctx, query, user.Email, user.Provider, user.ExternalID).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Role,
	)

	if err != nil {
		return fmt.Errorf("failed to create OAuth user: %w", err)
	}

	return nil
}

// GetUserByProviderAndExternalID retrieves a user by provider and external ID
func (r *UserRepository) GetUserByProviderAndExternalID(ctx context.Context, provider, externalID string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, provider, external_id, role, created_at, last_login_at
			FROM users
			WHERE provider = $1 AND external_id = $2
		`

	var user models.User
	err := r.db.Pool.QueryRow(ctx, query, provider, externalID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Provider,
		&user.ExternalID,
		&user.Role,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by provider and external ID: %w", err)
	}

	return &user, nil
}

// UpdateUser updates user fields (used for guest promotion)
func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, name = $2, avatar_key = $3, provider = $4, external_id = $5, role = $6
		WHERE id = $7
		RETURNING id, email, name, avatar_key, password_hash, provider, external_id, role, created_at, last_login_at
	`

	var updatedUser models.User
	err := r.db.Pool.QueryRow(ctx, query, user.Email, user.Name, user.AvatarKey, user.Provider, user.ExternalID, user.Role, user.ID).Scan(
		&updatedUser.ID,
		&updatedUser.Email,
		&updatedUser.Name,
		&updatedUser.AvatarKey,
		&updatedUser.PasswordHash,
		&updatedUser.Provider,
		&updatedUser.ExternalID,
		&updatedUser.Role,
		&updatedUser.CreatedAt,
		&updatedUser.LastLoginAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID int, lastLogin time.Time) error {
	query := `
		UPDATE users
		SET last_login_at = $1
		WHERE id = $2
	`

	result, err := r.db.Pool.Exec(ctx, query, lastLogin, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// LinkOAuthAccount links an OAuth account to an existing user
func (r *UserRepository) LinkOAuthAccount(ctx context.Context, userID int, provider, externalID string) error {
	query := `
		UPDATE users
		SET provider = $1, external_id = $2
		WHERE id = $3
	`

	result, err := r.db.Pool.Exec(ctx, query, provider, externalID, userID)
	if err != nil {
		return fmt.Errorf("failed to link OAuth account: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserProfile updates user's name and avatar_key
func (r *UserRepository) UpdateUserProfile(ctx context.Context, userID int, name, avatarURL *string) error {
	query := `
		UPDATE users 
		SET name = $2, avatar_key = $3
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, userID, name, avatarURL)
	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdatePlayerState updates user's player state (optimized for frequent calls)
func (r *UserRepository) UpdatePlayerState(ctx context.Context, userID int, trackID string, position float64, volume int) error {
	query := `
		UPDATE users 
		SET last_track_id = $2, last_position = $3, volume_preference = $4
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, userID, trackID, position, volume)
	if err != nil {
		return fmt.Errorf("failed to update player state: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetUserWithLastTrack retrieves a user by ID with last track details (using JOIN)
func (r *UserRepository) GetUserWithLastTrack(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT 
			u.id, u.email, u.name, u.avatar_key, u.password_hash, u.provider, u.external_id, u.role,
			u.last_track_id, u.last_position, u.volume_preference, u.created_at, u.last_login_at,
			t.id as track_id, t.title, t.artist, t.album_id, t.duration_seconds, t.audio_file_key,
			t.plays_count, t.likes_count, t.created_at as track_created_at
		FROM users u
		LEFT JOIN tracks t ON u.last_track_id = t.id
		WHERE u.id = $1
	`

	var user models.User
	var lastTrack models.Track
	var trackID, trackTitle, trackArtist, trackAlbumID, trackAudioKey *string
	var trackDuration, trackPlays, trackLikes *int
	var trackCreatedAt *time.Time

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AvatarKey,
		&user.PasswordHash,
		&user.Provider,
		&user.ExternalID,
		&user.Role,
		&user.LastTrackID,
		&user.LastPosition,
		&user.VolumePreference,
		&user.CreatedAt,
		&user.LastLoginAt,
		&trackID,
		&trackTitle,
		&trackArtist,
		&trackAlbumID,
		&trackDuration,
		&trackAudioKey,
		&trackPlays,
		&trackLikes,
		&trackCreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user with last track: %w", err)
	}

	// Log successful scan for debugging
	fmt.Printf("DEBUG: Successfully scanned user %d, email: %v, track: %v\n", user.ID, user.Email, trackID)

	// If user has a last track, populate the LastTrack field
	if trackID != nil && *trackID != "" {
		lastTrack.ID = *trackID
		lastTrack.Title = *trackTitle
		if trackArtist != nil {
			lastTrack.Artist = trackArtist
		}
		if trackAlbumID != nil {
			lastTrack.AlbumID = *trackAlbumID
		}
		lastTrack.DurationSeconds = *trackDuration
		lastTrack.AudioFileKey = *trackAudioKey
		lastTrack.PlaysCount = *trackPlays
		lastTrack.LikesCount = *trackLikes
		lastTrack.CreatedAt = *trackCreatedAt

		user.LastTrack = &lastTrack
	}

	return &user, nil
}

// TrackExists checks if a track exists by ID
func (r *UserRepository) TrackExists(ctx context.Context, trackID string) (bool, error) {
	query := `SELECT 1 FROM tracks WHERE id = $1`

	var exists int
	err := r.db.Pool.QueryRow(ctx, query, trackID).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if track exists: %w", err)
	}

	return true, nil
}
