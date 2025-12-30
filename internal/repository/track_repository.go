package repository

import (
	"context"
	"fmt"
	"time"

	"koteyye_music_be/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TrackRepository struct {
	db *DB
}

func NewTrackRepository(db *DB) *TrackRepository {
	return &TrackRepository{db: db}
}

// CreateTrack creates a new track in the database with album association
func (r *TrackRepository) CreateTrack(ctx context.Context, track *models.Track) error {
	query := `
		INSERT INTO tracks (user_id, album_id, title, artist, duration_seconds, s3_audio_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		track.UserID,
		track.AlbumID,
		track.Title,
		track.Artist,
		track.DurationSeconds,
		track.S3AudioKey,
	).Scan(
		&track.ID,
		&track.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	return nil
}

// GetTrackByID retrieves a track by its ID with album info
func (r *TrackRepository) GetTrackByID(ctx context.Context, id string) (*models.Track, error) {
	query := `
		SELECT t.id, t.user_id, t.album_id, t.title, t.artist, t.duration_seconds, 
		       t.s3_audio_key, t.plays_count, t.likes_count, t.created_at
		FROM tracks t
		WHERE t.id = $1
	`

	var track models.Track
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&track.ID,
		&track.UserID,
		&track.AlbumID,
		&track.Title,
		&track.Artist,
		&track.DurationSeconds,
		&track.S3AudioKey,
		&track.PlaysCount,
		&track.LikesCount,
		&track.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("track not found")
		}
		return nil, fmt.Errorf("failed to get track by ID: %w", err)
	}

	return &track, nil
}

// ListTracksWithAlbumInfo returns a paginated list of tracks with album info for frontend with optional genre filtering
func (r *TrackRepository) ListTracksWithAlbumInfo(ctx context.Context, limit, offset int, userID int, genreFilter string) ([]models.TrackResponse, error) {
	var query string
	var args []interface{}

	if userID != 0 {
		// For authenticated users, include like status
		query = `
			SELECT 
				t.id, t.title, t.duration_seconds, t.plays_count, t.likes_count, t.s3_audio_key,
				a.id as album_id, a.title as album_title, a.cover_image_key,
				COALESCE(t.artist, a.artist) as final_artist,
				a.release_date, a.genre,
				t.created_at,
				EXISTS(SELECT 1 FROM track_likes tl WHERE tl.track_id = t.id AND tl.user_id = $4) as is_liked
			FROM tracks t
			JOIN albums a ON t.album_id = a.id
			WHERE ($3 = '' OR a.genre = $3)
			ORDER BY t.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset, genreFilter, userID}
	} else {
		// For unauthenticated users, no like status
		query = `
			SELECT 
				t.id, t.title, t.duration_seconds, t.plays_count, t.likes_count, t.s3_audio_key,
				a.id as album_id, a.title as album_title, a.cover_image_key,
				COALESCE(t.artist, a.artist) as final_artist,
				a.release_date, a.genre,
				t.created_at, false as is_liked
			FROM tracks t
			JOIN albums a ON t.album_id = a.id
			WHERE ($3 = '' OR a.genre = $3)
			ORDER BY t.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset, genreFilter}
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks with album info: %w", err)
	}
	defer rows.Close()

	var tracks []models.TrackResponse
	for rows.Next() {
		var track models.TrackResponse
		var albumID string
		var releaseDate time.Time
		err := rows.Scan(
			&track.ID,
			&track.Title,
			&track.DurationSeconds,
			&track.PlaysCount,
			&track.LikesCount,
			&track.AudioFileKey,
			&albumID,
			&track.AlbumTitle,
			&track.CoverImageKey,
			&track.ArtistName,
			&releaseDate,
			&track.Genre,
			&track.CreatedAt,
			&track.IsLiked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		// Format release date and set album ID
		track.ReleaseDate = releaseDate.Format("2006-01-02")
		track.AlbumID = albumID
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTracksByUserID returns all tracks for a specific user with album info
func (r *TrackRepository) GetTracksByUserID(ctx context.Context, userID int) ([]models.TrackResponse, error) {
	query := `
		SELECT 
			t.id, t.title, t.duration_seconds, t.plays_count, t.likes_count, t.s3_audio_key,
			a.id as album_id, a.title as album_title, a.cover_image_key,
			COALESCE(t.artist, a.artist) as final_artist,
			a.release_date, a.genre,
			t.created_at, false as is_liked
		FROM tracks t
		JOIN albums a ON t.album_id = a.id
		WHERE t.user_id = $1
		ORDER BY t.created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks by user ID: %w", err)
	}
	defer rows.Close()

	var tracks []models.TrackResponse
	for rows.Next() {
		var track models.TrackResponse
		var albumID string
		var releaseDate time.Time
		err := rows.Scan(
			&track.ID,
			&track.Title,
			&track.DurationSeconds,
			&track.PlaysCount,
			&track.LikesCount,
			&track.AudioFileKey,
			&albumID,
			&track.AlbumTitle,
			&track.CoverImageKey,
			&track.ArtistName,
			&releaseDate,
			&track.Genre,
			&track.CreatedAt,
			&track.IsLiked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		// Format release date and set album ID
		track.ReleaseDate = releaseDate.Format("2006-01-02")
		track.AlbumID = albumID
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTrackWithAlbumInfo retrieves a track by ID with album info and like status
func (r *TrackRepository) GetTrackWithAlbumInfo(ctx context.Context, trackID string, userID int) (*models.TrackResponse, error) {
	var query string
	var args []interface{}

	if userID != 0 {
		query = `
			SELECT 
				t.id, t.title, t.duration_seconds, t.plays_count, t.likes_count, t.s3_audio_key,
				a.id as album_id, a.title as album_title, a.cover_image_key,
				COALESCE(t.artist, a.artist) as final_artist,
				a.release_date, a.genre,
				t.created_at,
				EXISTS(SELECT 1 FROM track_likes tl WHERE tl.track_id = t.id AND tl.user_id = $2) as is_liked
			FROM tracks t
			JOIN albums a ON t.album_id = a.id
			WHERE t.id = $1
		`
		args = []interface{}{trackID, userID}
	} else {
		query = `
			SELECT 
				t.id, t.title, t.duration_seconds, t.plays_count, t.likes_count, t.s3_audio_key,
				a.id as album_id, a.title as album_title, a.cover_image_key,
				COALESCE(t.artist, a.artist) as final_artist,
				a.release_date, a.genre,
				t.created_at, false as is_liked
			FROM tracks t
			JOIN albums a ON t.album_id = a.id
			WHERE t.id = $1
		`
		args = []interface{}{trackID}
	}

	var track models.TrackResponse
	var albumID string
	var releaseDate time.Time
	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(
		&track.ID,
		&track.Title,
		&track.DurationSeconds,
		&track.PlaysCount,
		&track.LikesCount,
		&track.AudioFileKey,
		&albumID,
		&track.AlbumTitle,
		&track.CoverImageKey,
		&track.ArtistName,
		&releaseDate,
		&track.Genre,
		&track.CreatedAt,
		&track.IsLiked,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("track not found")
		}
		return nil, fmt.Errorf("failed to get track with album info: %w", err)
	}

	// Format release date and set album ID
	track.ReleaseDate = releaseDate.Format("2006-01-02")
	track.AlbumID = albumID

	return &track, nil
}

// DeleteTrack deletes a track by its ID
func (r *TrackRepository) DeleteTrack(ctx context.Context, id string) error {
	query := `DELETE FROM tracks WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// UpdateTrack updates track metadata
func (r *TrackRepository) UpdateTrack(ctx context.Context, id string, track *models.Track) error {
	query := `
		UPDATE tracks
		SET title = $1, artist = $2
		WHERE id = $3
	`

	result, err := r.db.Pool.Exec(ctx, query,
		track.Title,
		track.Artist,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// ToggleLike toggles a like for a track (like if not liked, unlike if liked)
// Returns (isLiked, newLikesCount, error)
func (r *TrackRepository) ToggleLike(ctx context.Context, userID int, trackID string) (bool, int, error) {
	// Start transaction
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if track exists and get current likes count
	var currentLikesCount int
	checkQuery := `SELECT likes_count FROM tracks WHERE id = $1`
	err = tx.QueryRow(ctx, checkQuery, trackID).Scan(&currentLikesCount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, 0, fmt.Errorf("track not found")
		}
		return false, 0, fmt.Errorf("failed to check track: %w", err)
	}

	// Check if user already liked the track
	var existingLike int
	checkLikeQuery := `SELECT 1 FROM track_likes WHERE user_id = $1 AND track_id = $2`
	err = tx.QueryRow(ctx, checkLikeQuery, userID, trackID).Scan(&existingLike)

	isLiked := false
	var newLikesCount int

	if err == pgx.ErrNoRows {
		// User hasn't liked the track yet - add like
		insertLikeQuery := `
			INSERT INTO track_likes (user_id, track_id)
			VALUES ($1, $2)
		`
		_, err = tx.Exec(ctx, insertLikeQuery, userID, trackID)
		if err != nil {
			return false, 0, fmt.Errorf("failed to insert like: %w", err)
		}

		// Increment likes count
		updateCountQuery := `
			UPDATE tracks
			SET likes_count = likes_count + 1
			WHERE id = $1
			RETURNING likes_count
		`
		err = tx.QueryRow(ctx, updateCountQuery, trackID).Scan(&newLikesCount)
		if err != nil {
			return false, 0, fmt.Errorf("failed to update likes count: %w", err)
		}
		isLiked = true
	} else if err != nil {
		return false, 0, fmt.Errorf("failed to check existing like: %w", err)
	} else {
		// User already liked the track - remove like
		deleteLikeQuery := `
			DELETE FROM track_likes
			WHERE user_id = $1 AND track_id = $2
		`
		result, err := tx.Exec(ctx, deleteLikeQuery, userID, trackID)
		if err != nil {
			return false, 0, fmt.Errorf("failed to delete like: %w", err)
		}
		if result.RowsAffected() == 0 {
			return false, 0, fmt.Errorf("like not found")
		}

		// Decrement likes count
		updateCountQuery := `
			UPDATE tracks
			SET likes_count = likes_count - 1
			WHERE id = $1
			RETURNING likes_count
		`
		err = tx.QueryRow(ctx, updateCountQuery, trackID).Scan(&newLikesCount)
		if err != nil {
			return false, 0, fmt.Errorf("failed to update likes count: %w", err)
		}
		isLiked = false
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return false, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return isLiked, newLikesCount, nil
}

// IncrementPlays atomically increments the play count for a track
func (r *TrackRepository) IncrementPlays(ctx context.Context, trackID string) error {
	query := `
		UPDATE tracks
		SET plays_count = plays_count + 1
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, trackID)
	if err != nil {
		return fmt.Errorf("failed to increment plays: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("track not found")
	}

	return nil
}

// GetUserLikedTrackIDs returns a list of track IDs liked by the user
func (r *TrackRepository) GetUserLikedTrackIDs(ctx context.Context, userID int) ([]string, error) {
	query := `
		SELECT track_id
		FROM track_likes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user liked track IDs: %w", err)
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil {
			return nil, fmt.Errorf("failed to scan track ID: %w", err)
		}
		trackIDs = append(trackIDs, trackID)
	}

	return trackIDs, nil
}

// GetTrackStats returns play and like counts for a track
func (r *TrackRepository) GetTrackStats(ctx context.Context, trackID string) (*models.TrackStats, error) {
	query := `
		SELECT plays_count, likes_count
		FROM tracks
		WHERE id = $1
	`

	var stats models.TrackStats
	err := r.db.Pool.QueryRow(ctx, query, trackID).Scan(
		&stats.PlaysCount,
		&stats.LikesCount,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("track not found")
		}
		return nil, fmt.Errorf("failed to get track stats: %w", err)
	}

	return &stats, nil
}

// CountTracks returns the total number of tracks
func (r *TrackRepository) CountTracks(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tracks`

	err := r.db.Pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tracks: %w", err)
	}

	return count, nil
}

// ParseTrackID validates and parses a track ID from string
func ParseTrackID(id string) (uuid.UUID, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid track ID format: %w", err)
	}
	return parsedID, nil
}

// Legacy compatibility methods - to be removed after migration

// ListTracks - deprecated, use ListTracksWithAlbumInfo
func (r *TrackRepository) ListTracks(ctx context.Context, limit, offset int) ([]models.Track, error) {
	return nil, fmt.Errorf("deprecated method - use ListTracksWithAlbumInfo")
}

// ListTracksWithLikeStatus - deprecated, use ListTracksWithAlbumInfo
func (r *TrackRepository) ListTracksWithLikeStatus(ctx context.Context, limit, offset int, userID int) ([]models.Track, error) {
	return nil, fmt.Errorf("deprecated method - use ListTracksWithAlbumInfo")
}

// GetUserTracksWithLikeStatus - deprecated, use GetTracksByUserID
func (r *TrackRepository) GetUserTracksWithLikeStatus(ctx context.Context, userID int) ([]models.Track, error) {
	return nil, fmt.Errorf("deprecated method - use GetTracksByUserID")
}

// GetTrackWithLikeStatus - deprecated, use GetTrackWithAlbumInfo
func (r *TrackRepository) GetTrackWithLikeStatus(ctx context.Context, trackID string, userID int) (*models.Track, error) {
	return nil, fmt.Errorf("deprecated method - use GetTrackWithAlbumInfo")
}

// GetTracksWithZeroDuration returns tracks that have duration_seconds = 0
func (r *TrackRepository) GetTracksWithZeroDuration(ctx context.Context) ([]models.Track, error) {
	query := `
		SELECT id, user_id, album_id, title, artist, duration_seconds, s3_audio_key, 
			   plays_count, likes_count, created_at
		FROM tracks 
		WHERE duration_seconds = 0 OR duration_seconds IS NULL
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks with zero duration: %w", err)
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var track models.Track
		err := rows.Scan(
			&track.ID,
			&track.UserID,
			&track.AlbumID,
			&track.Title,
			&track.Artist,
			&track.DurationSeconds,
			&track.S3AudioKey,
			&track.PlaysCount,
			&track.LikesCount,
			&track.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// UpdateTrackDuration updates the duration_seconds field for a track
func (r *TrackRepository) UpdateTrackDuration(ctx context.Context, trackID string, durationSeconds int) error {
	query := "UPDATE tracks SET duration_seconds = $2 WHERE id = $1"
	result, err := r.db.Pool.Exec(ctx, query, trackID, durationSeconds)
	if err != nil {
		return fmt.Errorf("failed to update track duration: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return fmt.Errorf("track not found: %s", trackID)
	}
	
	return nil
}
