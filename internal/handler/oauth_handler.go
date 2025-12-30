package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"koteyye_music_be/internal/middleware"
	"koteyye_music_be/internal/service"
)

type OAuthHandler struct {
	oauthService *service.OAuthService
	logger       *slog.Logger
}

func NewOAuthHandler(oauthService *service.OAuthService, log *slog.Logger) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		logger:       log,
	}
}

// OAuthLogin handles OAuth login initiation
// @Summary OAuth Login Initiation
// @Tags oauth
// @Param provider path string true "OAuth Provider" Enums(google, yandex) Example(google)
// @Success 307 "Temporary Redirect to OAuth provider"
// @Failure 400 {object} map[string]string "Bad request - unsupported provider"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/auth/{provider}/login [get]
func (h *OAuthHandler) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Path[len("/api/auth/") : len(r.URL.Path)-len("/login")]

	if provider != "google" && provider != "yandex" {
		h.logger.Error("Unsupported OAuth provider", "provider", provider)
		http.Error(w, "Unsupported OAuth provider", http.StatusBadRequest)
		return
	}

	// Check if user is authenticated as guest
	// If so, we'll include guest ID in state for promotion
	var guestUserID *int
	if userID, ok := middleware.GetUserID(r.Context()); ok {
		// User is authenticated, check if they're a guest
		// Note: We'll need to fetch user role from DB or add role to context
		// For now, assume if authenticated via API, we'll pass the ID
		// The OAuth service will check if it's a valid guest during promotion
		guestUserID = &userID
		h.logger.Info("Guest attempting OAuth login", "guest_user_id", userID)
	}

	authURL, err := h.oauthService.GetAuthURL(provider, guestUserID)
	if err != nil {
		h.logger.Error("Failed to get OAuth URL", "provider", provider, "error", err)
		// Return the actual error message to help with configuration
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to OAuth provider
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallback handles OAuth callback from provider
// @Summary OAuth Callback
// @Tags oauth
// @Param provider path string true "OAuth Provider" Enums(google, yandex) Example(google)
// @Param code query string true "Authorization Code" Example(4/0AX4XfWhi_abc123xyz)
// @Param state query string false "State Parameter" Example(random_state_string)
// @Success 307 "Temporary Redirect to frontend with JWT token in query"
// @Failure 400 {object} map[string]string "Bad request - missing or invalid code"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/auth/{provider}/callback [get]
func (h *OAuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Path[len("/api/auth/") : len(r.URL.Path)-len("/callback")]

	if provider != "google" && provider != "yandex" {
		h.logger.Error("Unsupported OAuth provider", "provider", provider)
		http.Error(w, "Unsupported OAuth provider", http.StatusBadRequest)
		return
	}

	// Get code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		h.logger.Error("No code provided in OAuth callback", "provider", provider)
		http.Error(w, "No authorization code provided", http.StatusBadRequest)
		return
	}

	// Process OAuth callback
	token, user, err := h.oauthService.HandleCallback(r.Context(), provider, code, state)
	if err != nil {
		h.logger.Error("Failed to process OAuth callback", "provider", provider, "error", err)
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Build redirect URL with JWT token
	redirectURL, err := url.Parse(h.oauthService.GetFrontendURL())
	if err != nil {
		h.logger.Error("Failed to parse frontend URL", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Add token to query parameters
	query := redirectURL.Query()
	query.Set("token", token)
	query.Set("provider", provider)
	redirectURL.RawQuery = query.Encode()

	h.logger.Info("Redirecting to frontend with token",
		"user_id", user.ID,
		"provider", provider)

	// Redirect to frontend
	http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
}
