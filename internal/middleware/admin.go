package middleware

import (
	"net/http"

	"koteyye_music_be/internal/repository"
)

// RequireAdmin creates middleware that only allows admin users to access protected routes
func RequireAdmin(userRepo *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by AuthMiddleware)
			userID, ok := GetUserID(r.Context())
			if !ok {
				http.Error(w, `{"error":"User not authenticated"}`, http.StatusUnauthorized)
				return
			}

			// Get user from database to check role
			user, err := userRepo.GetUserByID(r.Context(), userID)
			if err != nil {
				http.Error(w, `{"error":"Failed to verify user permissions"}`, http.StatusInternalServerError)
				return
			}

			// Check if user has admin role
			if user.Role != "admin" {
				http.Error(w, `{"error":"Forbidden: Admin access required"}`, http.StatusForbidden)
				return
			}

			// User is admin, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth creates middleware that allows authenticated users (user, admin, guest) to access protected routes
func RequireAuth(userRepo *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by AuthMiddleware)
			userID, ok := GetUserID(r.Context())
			if !ok {
				http.Error(w, `{"error":"User not authenticated"}`, http.StatusUnauthorized)
				return
			}

			// Get user from database to check role
			user, err := userRepo.GetUserByID(r.Context(), userID)
			if err != nil {
				http.Error(w, `{"error":"Failed to verify user"}`, http.StatusInternalServerError)
				return
			}

			// Check if user has valid role (user, admin, or guest)
			if user.Role != "user" && user.Role != "admin" && user.Role != "guest" {
				http.Error(w, `{"error":"Forbidden: Invalid user role"}`, http.StatusForbidden)
				return
			}

			// User has valid role, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}
