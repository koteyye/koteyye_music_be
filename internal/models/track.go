package models

import "time"

type Track struct {
	ID              string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID          int       `json:"user_id" example:"1"`
	AlbumID         string    `json:"album_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Title           string    `json:"title" example:"Bohemian Rhapsody"`
	Artist          *string   `json:"artist,omitempty" example:"Queen"` // If NULL, uses album artist
	DurationSeconds int       `json:"duration_seconds" example:"354"`
	S3AudioKey      string    `json:"s3_audio_key" example:"albums/550e8400-e29b-41d4-a716-446655440001/550e8400-e29b-41d4-a716-446655440000.mp3"`
	PlaysCount      int       `json:"plays_count" example:"1250"`
	LikesCount      int       `json:"likes_count" example:"87"`
	IsLiked         bool      `json:"is_liked,omitempty" example:"true"` // Only in API responses
	CreatedAt       time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// TrackResponse represents track data with album info for frontend compatibility
type TrackResponse struct {
	ID              string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title           string    `json:"title" example:"Bohemian Rhapsody"`
	ArtistName      string    `json:"artist_name" example:"Queen"`
	AlbumID         string    `json:"album_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	AlbumTitle      string    `json:"album_title" example:"A Night at the Opera"`
	CoverURL        string    `json:"cover_url" example:"https://s3.amazonaws.com/bucket/albums/550e8400-e29b-41d4-a716-446655440001/cover.jpg"`
	CoverImageKey   string    `json:"-"` // Internal field for service layer, not exposed to frontend
	AudioURL        string    `json:"audio_url" example:"https://s3.amazonaws.com/bucket/albums/550e8400-e29b-41d4-a716-446655440001/550e8400-e29b-41d4-a716-446655440000.mp3"`
	AudioFileKey    string    `json:"-"` // Internal field for service layer, not exposed to frontend
	ReleaseDate     string    `json:"release_date" example:"1975-11-21"`
	Genre           string    `json:"genre" example:"rock"`
	DurationSeconds int       `json:"duration_seconds" example:"354"`
	PlaysCount      int       `json:"plays_count" example:"1250"`
	LikesCount      int       `json:"likes_count" example:"87"`
	IsLiked         bool      `json:"is_liked,omitempty" example:"true"`
	CreatedAt       time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type TrackCreate struct {
	AlbumID string  `json:"album_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
	Title   string  `json:"title" validate:"required,min=1,max=255" example:"Bohemian Rhapsody"`
	Artist  *string `json:"artist,omitempty" validate:"max=255" example:"Queen"` // Optional override artist
}

type TrackLike struct {
	UserID  int    `json:"user_id" example:"1"`
	TrackID string `json:"track_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type TrackStats struct {
	PlaysCount int `json:"plays_count" example:"1250"`
	LikesCount int `json:"likes_count" example:"87"`
}

// TrackListResponse represents response for listing tracks with pagination
type TrackListResponse struct {
	Tracks     []TrackResponse `json:"tracks"`
	Pagination TrackPagination `json:"pagination"`
}

// TrackPagination represents pagination metadata
type TrackPagination struct {
	Page  int `json:"page" example:"1"`
	Limit int `json:"limit" example:"20"`
	Total int `json:"total" example:"42"`
}

// UserTracksResponse represents the response for user's tracks
type UserTracksResponse struct {
	Tracks []TrackResponse `json:"tracks"`
}

// ToggleLikeResponse represents the response for toggling track like
type ToggleLikeResponse struct {
	Liked      bool `json:"liked" example:"true"`
	LikesCount int  `json:"likes_count" example:"88"`
}

// GenreFilter represents filter for content by genre
type GenreFilter struct {
	Genre string `json:"genre,omitempty" example:"rock"`
}
