-- Complete Database Schema V2 with All Features
-- Includes: Album architecture, Player state, MinIO integration, Extended genres

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Drop existing tables if they exist (for clean setup)
DROP TABLE IF EXISTS track_likes CASCADE;
DROP TABLE IF EXISTS tracks CASCADE;
DROP TABLE IF EXISTS albums CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS schema_migrations CASCADE;

-- Users table with full player state support
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE,  -- NULL for guests
    password_hash VARCHAR(255), -- NULL for guests and OAuth users
    provider VARCHAR(50),       -- NULL for guests, 'local'/'google'/'yandex' for others
    external_id VARCHAR(255),   -- NULL for local users
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    name VARCHAR(255),          -- User display name (from OAuth or profile)
    avatar_url VARCHAR(500),    -- User avatar URL
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Player state fields with expanded functionality
    last_track_id UUID,         -- Last played track (no FK constraint to avoid circular dependency)
    last_position INTEGER DEFAULT 0,  -- Position in seconds where user left off
    volume_preference INTEGER DEFAULT 100, -- User's preferred volume (0-100)
    last_played_at TIMESTAMP,   -- When track was last played
    
    CONSTRAINT check_valid_role CHECK (role IN ('user', 'admin', 'guest')),
    CONSTRAINT check_valid_volume CHECK (volume_preference >= 0 AND volume_preference <= 100)
);

-- Albums table with expanded genre support
CREATE TABLE albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    release_date DATE NOT NULL,
    genre VARCHAR(50) NOT NULL CHECK (genre IN (
        'pop', 'rock', 'hip-hop', 'rap', 'indie', 'electronic', 'house', 'techno',
        'jazz', 'blues', 'classical', 'metal', 'punk', 'r-n-b', 'soul', 'folk',
        'reggae', 'country', 'latin', 'k-pop', 'soundtrack', 'lo-fi', 'chanson'
    )),
    cover_image_key VARCHAR(500) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tracks table with proper audio file key naming
CREATE TABLE tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    album_id UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    artist VARCHAR(255), -- Track-specific artist. If NULL, uses album artist
    duration_seconds INTEGER DEFAULT 0, -- Track duration extracted from audio file
    audio_file_key VARCHAR(500) NOT NULL, -- MinIO storage key for audio file
    plays_count BIGINT DEFAULT 0,
    likes_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Track likes table for Many-to-Many relationship
CREATE TABLE track_likes (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_track_likes_user_track PRIMARY KEY (user_id, track_id)
);

-- Performance indexes for users
CREATE INDEX idx_users_provider ON users(provider);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_last_login_at ON users(last_login_at DESC);
CREATE INDEX idx_users_last_track ON users(last_track_id);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Performance indexes for albums
CREATE INDEX idx_albums_genre ON albums(genre);
CREATE INDEX idx_albums_artist ON albums(artist);
CREATE INDEX idx_albums_release_date ON albums(release_date DESC);
CREATE INDEX idx_albums_created_at ON albums(created_at DESC);

-- Performance indexes for tracks
CREATE INDEX idx_tracks_user_id ON tracks(user_id);
CREATE INDEX idx_tracks_album_id ON tracks(album_id);
CREATE INDEX idx_tracks_created_at ON tracks(created_at DESC);
CREATE INDEX idx_tracks_plays_count ON tracks(plays_count DESC);
CREATE INDEX idx_tracks_likes_count ON tracks(likes_count DESC);

-- Performance indexes for track likes
CREATE INDEX idx_track_likes_user_id ON track_likes(user_id);
CREATE INDEX idx_track_likes_track_id ON track_likes(track_id);
CREATE INDEX idx_track_likes_created_at ON track_likes(created_at DESC);

-- Unique constraint for OAuth users
CREATE UNIQUE INDEX idx_users_provider_external_id 
    ON users(provider, external_id) 
    WHERE provider IS NOT NULL AND external_id IS NOT NULL;

-- Add updated_at trigger for albums
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_albums_updated_at BEFORE UPDATE ON albums
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Schema migrations table (create after dropping everything)
CREATE TABLE schema_migrations (
    name VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert this migration as applied
INSERT INTO schema_migrations (name) VALUES ('005_complete_schema_v2.sql');

-- Comments for documentation
COMMENT ON TABLE users IS 'User accounts including regular users, admins, and guests with player state';
COMMENT ON TABLE albums IS 'Album metadata including cover, artist, genre and release info';
COMMENT ON TABLE tracks IS 'Individual tracks belonging to albums with audio files';
COMMENT ON TABLE track_likes IS 'User likes for tracks (many-to-many relationship)';

COMMENT ON COLUMN users.provider IS 'Authentication provider: local, google, yandex, or NULL for guests';
COMMENT ON COLUMN users.role IS 'User role: user, admin, or guest';
COMMENT ON COLUMN users.last_track_id IS 'Last played track ID for resuming playback';
COMMENT ON COLUMN users.last_position IS 'Position in seconds where user left off in last track';
COMMENT ON COLUMN users.volume_preference IS 'User preferred volume level 0-100';

COMMENT ON COLUMN albums.cover_image_key IS 'Path to album cover image in MinIO storage';
COMMENT ON COLUMN albums.genre IS 'Music genre from expanded predefined list (23 genres)';

COMMENT ON COLUMN tracks.artist IS 'Track-specific artist. If NULL, uses album artist';
COMMENT ON COLUMN tracks.album_id IS 'Reference to parent album containing metadata';
COMMENT ON COLUMN tracks.audio_file_key IS 'MinIO storage key for audio file';
COMMENT ON COLUMN tracks.duration_seconds IS 'Track duration in seconds extracted from audio metadata';
COMMENT ON COLUMN tracks.likes_count IS 'Denormalized counter for likes. Updated by trigger or application logic.';
COMMENT ON COLUMN tracks.plays_count IS 'Denormalized counter for plays. Updated by application logic on track playback.';