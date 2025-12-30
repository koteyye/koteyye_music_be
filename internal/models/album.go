package models

import (
	"strings"
	"time"
)

type Album struct {
	ID            string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title         string    `json:"title" example:"A Night at the Opera"`
	Artist        string    `json:"artist" example:"Queen"`
	ReleaseDate   time.Time `json:"release_date" example:"1975-11-21"`
	Genre         string    `json:"genre" example:"rock"`
	CoverImageKey string    `json:"cover_image_key" example:"albums/550e8400-e29b-41d4-a716-446655440000/cover.jpg"`
	CoverURL      string    `json:"cover_url,omitempty" example:"https://s3.amazonaws.com/bucket/albums/550e8400-e29b-41d4-a716-446655440000/cover.jpg"`
	CreatedAt     time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type AlbumCreate struct {
	Title       string `json:"title" validate:"required,min=1,max=255" example:"A Night at the Opera"`
	Artist      string `json:"artist" validate:"required,min=1,max=255" example:"Queen"`
	ReleaseDate string `json:"release_date" validate:"required" example:"1975-11-21"`
	Genre       string `json:"genre" validate:"required" example:"rock"`
}

type AlbumResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title       string    `json:"title" example:"A Night at the Opera"`
	Artist      string    `json:"artist" example:"Queen"`
	ReleaseDate string    `json:"release_date" example:"1975-11-21"`
	Genre       string    `json:"genre" example:"rock"`
	CoverURL    string    `json:"cover_url" example:"https://s3.amazonaws.com/bucket/albums/550e8400-e29b-41d4-a716-446655440000/cover.jpg"`
	Year        int       `json:"year" example:"1975"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type AlbumDetail struct {
	Album  AlbumResponse   `json:"album"`
	Tracks []TrackResponse `json:"tracks"`
}

// AllowedGenres represents valid music genres (lowercase keys)
var AllowedGenres = []string{
	"pop", "rock", "hip-hop", "rap", "indie", "electronic", "house", "techno",
	"jazz", "blues", "classical", "metal", "punk", "r-n-b", "soul", "folk",
	"reggae", "country", "latin", "k-pop", "soundtrack", "lo-fi", "chanson",
}

// IsValidGenre checks if the genre is in the allowed list (case-insensitive)
func IsValidGenre(genre string) bool {
	genreLower := strings.ToLower(genre)
	for _, allowed := range AllowedGenres {
		if genreLower == allowed {
			return true
		}
	}
	return false
}
