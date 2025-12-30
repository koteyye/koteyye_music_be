package models

import "time"

type User struct {
	ID               int        `json:"id" example:"1"`
	Email            *string    `json:"email,omitempty" example:"user@example.com"`                             // NULL для гостей
	Name             *string    `json:"name,omitempty" example:"John Doe"`                                      // NULL если не указано
	AvatarURL        *string    `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`          // NULL если нет аватара
	PasswordHash     *string    `json:"-" example:""`                                                           // NULL для гостей и OAuth пользователей
	Provider         *string    `json:"provider,omitempty" example:"local"`                                     // NULL для гостей, 'local', 'google', 'yandex'
	ExternalID       *string    `json:"external_id,omitempty" example:""`                                       // NULL для локальных пользователей
	Role             string     `json:"role" example:"user"`                                                    // 'user', 'admin', 'guest'
	LastTrackID      *string    `json:"last_track_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"` // NULL если нет последнего трека
	LastPosition     float64    `json:"last_position" example:"45.5"`                                           // Позиция в секундах
	VolumePreference int        `json:"volume_preference" example:"80"`                                         // Громкость 0-100
	LastTrack        *Track     `json:"last_track,omitempty"`                                                   // Полный объект последнего трека (JOIN)
	LastLoginAt      *time.Time `json:"last_login_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CreatedAt        time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email" example:"newuser@example.com"`
	Password string `json:"password" validate:"required,min=6" example:"password123"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email" example:"user@example.com"`
	Password string `json:"password" validate:"required" example:"password123"`
}

type AuthResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZXhwIjoxNzA1MzQzODAwfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
	User  User   `json:"user"`
}

// LoginResponse is an alias for AuthResponse for Swagger compatibility
type LoginResponse = AuthResponse

// GuestResponse represents response for guest login
type GuestResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZXhwIjoxNzA1MzQzODAwfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
	User  User   `json:"user"`
}

// OAuthUserInfo represents user info from OAuth providers
type OAuthUserInfo struct {
	Email      string
	Name       string
	AvatarURL  string
	ExternalID string
	Provider   string
}

// UserProfileResponse represents user profile data for /me endpoint
type UserProfileResponse struct {
	ID               int        `json:"id" example:"1"`
	Email            *string    `json:"email,omitempty" example:"user@example.com"`
	Name             *string    `json:"name,omitempty" example:"John Doe"`
	AvatarURL        *string    `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Provider         *string    `json:"provider,omitempty" example:"local"`
	Role             string     `json:"role" example:"user"`
	LastTrackID      *string    `json:"last_track_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	LastPosition     float64    `json:"last_position" example:"45.5"`
	VolumePreference int        `json:"volume_preference" example:"80"`
	LastTrack        *Track     `json:"last_track,omitempty"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CreatedAt        time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// UpdateProfileRequest represents profile update data
type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty" example:"John Doe"`
	AvatarURL *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
}

// PlayerStateRequest represents player state update data
type PlayerStateRequest struct {
	TrackID  string  `json:"track_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Position float64 `json:"position" validate:"min=0" example:"45.5"`
	Volume   int     `json:"volume" validate:"min=0,max=100" example:"80"`
}
