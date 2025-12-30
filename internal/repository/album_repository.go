package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"koteyye_music_be/internal/models"
)

type AlbumRepository struct {
	db *pgxpool.Pool
}

func NewAlbumRepository(db *pgxpool.Pool) *AlbumRepository {
	return &AlbumRepository{db: db}
}

func (r *AlbumRepository) Create(ctx context.Context, album *models.Album) error {
	query := `
		INSERT INTO albums (id, title, artist, release_date, genre, cover_image_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		album.ID,
		album.Title,
		album.Artist,
		album.ReleaseDate,
		album.Genre,
		album.CoverImageKey,
		album.CreatedAt,
		album.UpdatedAt,
	)
	return err
}

func (r *AlbumRepository) GetByID(ctx context.Context, id string) (*models.Album, error) {
	query := `
		SELECT id, title, artist, release_date, genre, cover_image_key, created_at, updated_at
		FROM albums
		WHERE id = $1
	`
	var album models.Album
	err := r.db.QueryRow(ctx, query, id).Scan(
		&album.ID,
		&album.Title,
		&album.Artist,
		&album.ReleaseDate,
		&album.Genre,
		&album.CoverImageKey,
		&album.CreatedAt,
		&album.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &album, nil
}

func (r *AlbumRepository) GetAll(ctx context.Context, limit, offset int, genreFilter string) ([]models.Album, error) {
	query := `
		SELECT id, title, artist, release_date, genre, cover_image_key, created_at, updated_at
		FROM albums
		WHERE ($3 = '' OR genre = $3)
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset, genreFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var albums []models.Album
	for rows.Next() {
		var album models.Album
		err := rows.Scan(
			&album.ID,
			&album.Title,
			&album.Artist,
			&album.ReleaseDate,
			&album.Genre,
			&album.CoverImageKey,
			&album.CreatedAt,
			&album.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		albums = append(albums, album)
	}
	return albums, rows.Err()
}

func (r *AlbumRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM albums WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *AlbumRepository) GetAlbumWithTracks(ctx context.Context, albumID string) (*models.AlbumDetail, error) {
	// Get album info
	album, err := r.GetByID(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}

	// Get tracks for this album
	tracksQuery := `
		SELECT 
			t.id, t.title, t.duration_seconds, t.plays_count, t.likes_count, t.s3_audio_key,
			a.id as album_id, a.title as album_title, a.cover_image_key,
			COALESCE(t.artist, a.artist) as final_artist,
			a.release_date, a.genre,
			t.created_at, false as is_liked
		FROM tracks t
		JOIN albums a ON t.album_id = a.id
		WHERE t.album_id = $1
		ORDER BY t.created_at ASC
	`

	rows, err := r.db.Query(ctx, tracksQuery, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks: %w", err)
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

	// Convert release date to year
	year := album.ReleaseDate.Year()

	albumResponse := models.AlbumResponse{
		ID:          album.ID,
		Title:       album.Title,
		Artist:      album.Artist,
		ReleaseDate: album.ReleaseDate.Format("2006-01-02"),
		Genre:       album.Genre,
		Year:        year,
		CreatedAt:   album.CreatedAt,
	}

	return &models.AlbumDetail{
		Album:  albumResponse,
		Tracks: tracks,
	}, nil
}

func (r *AlbumRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM albums").Scan(&count)
	return count, err
}
