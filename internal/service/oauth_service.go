package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/repository"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/yandex"
)

type OAuthService struct {
	userRepo     *repository.UserRepository
	authService  *AuthService
	logger       *slog.Logger
	googleConfig *oauth2.Config
	yandexConfig *oauth2.Config
	frontendURL  string
}

// Ensure AuthService methods can be called on pointer

// GoogleUserInfo represents user info from Google API
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// YandexUserInfo represents user info from Yandex API
type YandexUserInfo struct {
	ID            string   `json:"id"`
	Login         string   `json:"login"`
	Emails        []string `json:"emails"`
	DefaultEmail  string   `json:"default_email"`
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	RealName      string   `json:"real_name"`
	DefaultAvatar string   `json:"default_avatar_id"`
}

func NewOAuthService(
	userRepo *repository.UserRepository,
	authService *AuthService,
	googleClientID, googleClientSecret, googleRedirectURL string,
	yandexClientID, yandexClientSecret, yandexRedirectURL string,
	frontendURL string,
	logger *slog.Logger,
) *OAuthService {
	return &OAuthService{
		userRepo:    userRepo,
		authService: authService,
		logger:      logger,
		frontendURL: frontendURL,
		googleConfig: &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		yandexConfig: &oauth2.Config{
			ClientID:     yandexClientID,
			ClientSecret: yandexClientSecret,
			RedirectURL:  yandexRedirectURL,
			Scopes:       []string{"login:email"},
			Endpoint:     yandex.Endpoint,
		},
	}
}

// GetAuthURL generates OAuth authorization URL for the specified provider
// guestUserID is optional - if provided, guest will be promoted to user on OAuth login
func (s *OAuthService) GetAuthURL(provider string, guestUserID *int) (string, error) {
	var config *oauth2.Config
	var providerName string

	switch provider {
	case "google":
		config = s.googleConfig
		providerName = "Google"
		if config.ClientID == "" || config.ClientSecret == "" {
			return "", fmt.Errorf("Google OAuth credentials not configured. Please set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET environment variables")
		}
	case "yandex":
		config = s.yandexConfig
		providerName = "Yandex"
		if config.ClientID == "" || config.ClientSecret == "" {
			return "", fmt.Errorf("Yandex OAuth credentials not configured. Please set YANDEX_CLIENT_ID and YANDEX_CLIENT_SECRET environment variables")
		}
	default:
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Generate state with guest user ID for guest promotion
	// Format: "provider:guestUserID" or just "provider" if no guest
	var state string
	if guestUserID != nil {
		state = fmt.Sprintf("%s:guest:%d", provider, *guestUserID)
	} else {
		state = fmt.Sprintf("%s:guest:0", provider)
	}

	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	s.logger.Info("Generated OAuth URL",
		"provider", providerName,
		"guest_user_id", guestUserID,
		"redirect_url", config.RedirectURL,
		"client_id", config.ClientID,
		"url", authURL)

	return authURL, nil
}

// HandleCallback processes OAuth callback from provider
// Returns JWT token and user info for redirect to frontend
// Supports guest promotion: if state contains guest ID, guest will be promoted
func (s *OAuthService) HandleCallback(ctx context.Context, provider, code, state string) (string, *models.User, error) {
	var config *oauth2.Config
	var providerName string
	var userInfo *models.OAuthUserInfo

	switch provider {
	case "google":
		config = s.googleConfig
		providerName = "Google"
	case "yandex":
		config = s.yandexConfig
		providerName = "Yandex"
	default:
		return "", nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Exchange code for access token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		s.logger.Error("Failed to exchange code for token", "provider", providerName, "error", err)
		return "", nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from provider
	userInfo, err = s.getUserInfo(ctx, provider, token)
	if err != nil {
		s.logger.Error("Failed to get user info from provider", "provider", providerName, "error", err)
		return "", nil, fmt.Errorf("failed to get user info: %w", err)
	}

	userInfo.Provider = provider

	// Parse state to extract guest user ID
	guestUserID := s.parseStateForGuestID(state)
	if guestUserID > 0 {
		s.logger.Info("OAuth login with guest promotion", "provider", providerName, "guest_user_id", guestUserID)
	}

	// Find or create user (with guest promotion support)
	user, err := s.findOrCreateUser(ctx, userInfo, guestUserID)
	if err != nil {
		s.logger.Error("Failed to find or create user", "provider", providerName, "error", err)
		return "", nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Generate JWT token
	jwtToken, err := s.authService.GenerateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate JWT token", "user_id", user.ID, "error", err)
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("User authenticated via OAuth",
		"provider", providerName,
		"user_id", user.ID,
		"email", user.Email,
		"was_guest", guestUserID > 0)

	return jwtToken, user, nil
}

// parseStateForGuestID extracts guest user ID from OAuth state parameter
// State format: "provider:guest:userID" or "provider:guest:0"
// Returns guest user ID if valid (>0), otherwise 0
func (s *OAuthService) parseStateForGuestID(state string) int {
	if state == "" {
		return 0
	}

	// Simple parsing: extract last part after "guest:"
	// Expected format: "google:guest:123" or "yandex:guest:456"
	guestPrefix := "guest:"
	idx := len(state) - len(guestPrefix)
	if idx < 0 {
		idx = 0
	} else {
		idx = len(state) - idx
	}

	// Find guest: in state
	guestIdx := findSubstring(state, guestPrefix)
	if guestIdx == -1 {
		return 0
	}

	// Extract guest ID after "guest:"
	guestIDStr := state[guestIdx+len(guestPrefix):]

	// Parse guest ID
	var guestID int
	if _, err := fmt.Sscanf(guestIDStr, "%d", &guestID); err != nil {
		s.logger.Warn("Failed to parse guest ID from state", "state", state, "error", err)
		return 0
	}

	return guestID
}

// findSubstring finds the last occurrence of substring in string
func findSubstring(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetFrontendURL returns the frontend URL for OAuth redirect
func (s *OAuthService) GetFrontendURL() string {
	return s.frontendURL
}

// getUserInfo fetches user information from OAuth provider
func (s *OAuthService) getUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*models.OAuthUserInfo, error) {
	client := &http.Client{}
	var endpoint string

	switch provider {
	case "google":
		endpoint = "https://www.googleapis.com/oauth2/v2/userinfo"
	case "yandex":
		endpoint = "https://login.yandex.ru/info?format=json"
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	switch provider {
	case "google":
		return s.parseGoogleUserInfo(resp.Body)
	case "yandex":
		return s.parseYandexUserInfo(resp.Body)
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// parseGoogleUserInfo parses Google API response
func (s *OAuthService) parseGoogleUserInfo(body io.Reader) (*models.OAuthUserInfo, error) {
	var info GoogleUserInfo
	if err := json.NewDecoder(body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	if !info.VerifiedEmail {
		return nil, fmt.Errorf("email is not verified")
	}

	return &models.OAuthUserInfo{
		Email:      info.Email,
		Name:       info.Name,
		AvatarURL:  info.Picture,
		ExternalID: info.ID,
		Provider:   "google",
	}, nil
}

// parseYandexUserInfo parses Yandex API response
func (s *OAuthService) parseYandexUserInfo(body io.Reader) (*models.OAuthUserInfo, error) {
	var info YandexUserInfo
	if err := json.NewDecoder(body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode Yandex user info: %w", err)
	}

	// Use default_email if available, otherwise use first email
	email := info.DefaultEmail
	if email == "" && len(info.Emails) > 0 {
		email = info.Emails[0]
	}

	if email == "" {
		return nil, fmt.Errorf("email not found in Yandex response")
	}

	name := info.RealName
	if name == "" {
		name = info.FirstName
		if info.LastName != "" {
			if name != "" {
				name += " " + info.LastName
			} else {
				name = info.LastName
			}
		}
	}

	// Construct Yandex avatar URL if avatar ID exists
	var avatarURL string
	if info.DefaultAvatar != "" {
		avatarURL = fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-middle", info.DefaultAvatar)
	}

	return &models.OAuthUserInfo{
		Email:      email,
		Name:       name,
		AvatarURL:  avatarURL,
		ExternalID: info.ID,
		Provider:   "yandex",
	}, nil
}

// findOrCreateUser finds existing user by email or creates a new one
// Supports guest promotion: if guestUserID > 0, guest will be promoted instead of creating new user
func (s *OAuthService) findOrCreateUser(ctx context.Context, userInfo *models.OAuthUserInfo, guestUserID int) (*models.User, error) {
	// First, try to find user by email
	user, err := s.userRepo.GetUserByEmail(ctx, userInfo.Email)
	if err == nil && user != nil {
		// SCENARIO A: User already exists by email
		// Just login them - guest history is lost (expected behavior)
		s.logger.Info("User found by email, logging in", "email", userInfo.Email, "existing_user_id", user.ID)

		// If user was registered locally, we can link OAuth account
		if user.Provider != nil && *user.Provider == "local" {
			user.ExternalID = &userInfo.ExternalID
			user.Provider = &userInfo.Provider

			if err := s.userRepo.LinkOAuthAccount(ctx, user.ID, *user.Provider, *user.ExternalID); err != nil {
				s.logger.Warn("Failed to link OAuth account to existing user",
					"user_id", user.ID,
					"provider", userInfo.Provider,
					"error", err)
				// Continue anyway, just update user object
			} else {
				s.logger.Info("Linked OAuth account to existing user",
					"user_id", user.ID,
					"provider", userInfo.Provider)
			}
		} else if user.Provider != nil && *user.Provider != userInfo.Provider {
			// User exists but with different provider
			// This is a conflict situation - user with this email already exists via another OAuth provider
			return nil, fmt.Errorf("user already exists with provider: %s", *user.Provider)
		}

		// Update last login time
		now := time.Now()
		if err := s.userRepo.UpdateLastLogin(ctx, user.ID, now); err != nil {
			s.logger.Warn("Failed to update last login time", "user_id", user.ID, "error", err)
		}

		return user, nil
	}

	// User not found by email
	if guestUserID > 0 {
		// SCENARIO B: Guest promotion - upgrade existing guest to registered user
		s.logger.Info("Promoting guest to registered user", "guest_user_id", guestUserID, "email", userInfo.Email)

		response, err := s.authService.PromoteGuestToUser(ctx, guestUserID, userInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to promote guest: %w", err)
		}

		return &response.User, nil
	}

	// SCENARIO C: Clean login - create new user
	s.logger.Info("Creating new OAuth user", "email", userInfo.Email)
	return s.createOAuthUser(ctx, userInfo)
}

// createOAuthUser creates a new user from OAuth info
func (s *OAuthService) createOAuthUser(ctx context.Context, userInfo *models.OAuthUserInfo) (*models.User, error) {
	// Prepare optional fields
	var name *string
	if userInfo.Name != "" {
		name = &userInfo.Name
	}
	// TODO: Download avatarURL and convert to key
	_ = userInfo.AvatarURL

	user := &models.User{
		Email:      &userInfo.Email,
		Name:       name,
		AvatarKey:  nil, // TODO: Download and store OAuth avatars to MinIO
		Provider:   &userInfo.Provider,
		ExternalID: &userInfo.ExternalID,
		Role:       "user",
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	s.logger.Info("Created new OAuth user",
		"user_id", user.ID,
		"provider", userInfo.Provider,
		"email", userInfo.Email)

	return user, nil
}
