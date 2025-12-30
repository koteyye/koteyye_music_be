# Koteyye Music Backend - Development Guide

## Build/Test/Lint Commands
- **Build**: `go build -o /tmp/koteyye_music_be ./cmd/api`
- **Run**: `go run cmd/api/main.go`
- **Test**: `go test ./...` (currently no tests exist, needs implementation)
- **Test single package**: `go test -v ./internal/service`
- **Format**: `go fmt ./...`
- **Vet**: `go vet ./...`
- **Swagger docs**: `bash scripts/generate_swagger.sh`
- **Docker services**: `cd scripts && docker-compose up -d`

## Code Style Guidelines
- **Package naming**: Use short, lowercase names (auth, handler, service)
- **Struct naming**: PascalCase with descriptive names (AuthHandler, TrackService)
- **Function naming**: PascalCase for exported, camelCase for private
- **Import grouping**: stdlib, third-party, local packages with blank lines between
- **Error handling**: Always wrap errors with context using `fmt.Errorf("message: %w", err)`
- **Logging**: Use structured logging with slog (`logger.Info("message", "key", value)`)
- **Pointers**: Use pointers for optional fields in models (email *string for nullable DB fields)
- **JSON tags**: Include json tags with omitempty for optional fields
- **HTTP responses**: Use consistent error format `{"error": "message"}`
- **Context**: Always pass context as first parameter, use `ctx := r.Context()`
- **Validation**: Validate input at handler level before passing to service
- **Database**: Use pgx for PostgreSQL, wrap queries in transactions when needed
- **JWT**: Include user_id, email, role in token claims with 24h expiration
- **File organization**: Follow clean architecture - handler/service/repository/models
- **Constants**: Use typed constants for context keys and enums
- **Comments**: Use Swagger comments for API documentation
- **Security**: Never expose password hashes, validate JWT tokens properly

## API Endpoints
- **Auth**: All auth endpoints under `/api/auth/*` prefix
- **OAuth**: `GET /api/auth/{google|yandex}/login` for OAuth initiation (with avatar support)
- **User Profile**: `GET /api/users/me`, `PUT /api/users/me` - user profile management
- **Player State**: `POST /api/user/player-state` - update listening progress and volume
- **Public endpoints**: `GET /api/tracks`, `GET /api/tracks/{id}`, `GET /api/tracks/{id}/stream`, `GET /api/tracks/{id}/cover` - work without authentication
- **Authenticated endpoints**: `GET /api/tracks/my`, `POST /api/tracks/{id}/like` - require auth (including guest tokens)
- **Guest login**: `POST /api/auth/guest` creates temporary guest users
- **Account promotion**: OAuth flow can promote guest users to registered users with avatar/name

## Recent Fixes
- **Registration 500 Error**: Fixed logic in auth service - now properly detects existing users and returns 409 Conflict instead of 500
- **OAuth Routes**: Moved from `/auth/*` to `/api/auth/*` to match frontend expectations
- **Database Migration**: Fixed path resolution and constraint creation for guest mode support
- **OAuth Redirect URL**: Fixed port mismatch in .env file (8081 -> 8080)
- **Track Upload 500 Error**: Fixed FFmpeg file naming conflict - changed input/output paths to avoid overwriting same file
- **Track Upload Form Field**: Changed expected field from "image" to "cover" to match frontend
- **Missing Cover Endpoint**: Added `/api/tracks/{id}/cover` endpoint for serving track cover images

## New Features Added
- **User Profile API**: `/api/users/me` endpoint for getting/updating user profile
- **Avatar Support**: Automatic avatar retrieval from Google/Yandex OAuth providers
- **Profile Fields**: Added name and avatar_url fields to user model
- **OAuth Enhancements**: OAuth now saves user name and avatar during registration/promotion
- **Single Track Endpoint**: `/api/tracks/{id}` for deep linking to specific tracks with optional like status
- **Track Cover Images**: `/api/tracks/{id}/cover` endpoint for serving track cover images
- **Player State Management**: Store/sync user's listening progress across devices
  - Database fields: `last_track_id`, `last_position`, `volume_preference`
  - API endpoint: `POST /api/users/player-state` for updating state
- **Genre Filtering**: Filter content by music genre in public endpoints
  - `GET /api/tracks?genre=rock` - filter tracks by genre
  - `GET /api/albums?genre=rock` - filter albums by genre
  - 23 supported genres: pop, rock, hip-hop, rap, indie, electronic, house, techno, jazz, blues, classical, metal, punk, r-n-b, soul, folk, reggae, country, latin, k-pop, soundtrack, lo-fi, chanson
- **Audio Duration Extraction**: Automatic extraction of track duration from uploaded audio files
  - Uses FFprobe to extract metadata from audio files during upload
  - Supports all major audio formats (MP3, WAV, M4A, FLAC, etc.)
  - Duration stored as `duration_seconds` field in database and API responses
- **Enhanced Track API**: Added `album_id` field to track responses
  - All track endpoints now include `album_id` for easy album navigation
  - Enables deep linking from tracks to their parent albums
  - Profile responses include full last track details with JOIN optimization

## Known Issues
- Test files don't exist - create comprehensive test suite

## AI Assistant Guidelines
- **Language**: Communicate in Russian (общайся на русском языке)
- **Testing**: Do not write unit tests (не пиши unit-тесты)

## Development Workflow
1. Run `go fmt ./...` before commits
2. Generate swagger docs after API changes
3. Ensure database migrations run automatically on startup
4. Use structured logging throughout the application