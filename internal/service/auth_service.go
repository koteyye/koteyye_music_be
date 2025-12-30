package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"koteyye_music_be/internal/models"
	"koteyye_music_be/internal/repository"

	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	logger    *slog.Logger
}

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, log *slog.Logger) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		logger:    log,
	}
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	_, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		// User found, email already exists
		return nil, errors.New("user with this email already exists")
	}
	// User not found - this is expected for registration, continue

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", "error", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	email := req.Email
	passwordHash := string(hashedPassword)
	provider := "local"
	user := &models.User{
		Email:        &email,
		PasswordHash: &passwordHash,
		Provider:     &provider,
		Role:         "user", // Explicitly set role for registration
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token
	token, err := s.GenerateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("User registered successfully", "user_id", user.ID, "email", user.Email)

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Get user
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Warn("Login attempt with non-existent email", "email", req.Email)
		return nil, errors.New("invalid email or password")
	}

	// Check password
	if user.PasswordHash == nil {
		return nil, errors.New("user must login via OAuth")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		s.logger.Warn("Invalid password attempt", "user_id", user.ID, "email", user.Email)
		return nil, errors.New("invalid email or password")
	}

	// Generate token
	token, err := s.GenerateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Update last login timestamp
	now := time.Now()
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, now); err != nil {
		s.logger.Warn("Failed to update last login time", "user_id", user.ID, "error", err)
	}

	// Get user with last track details for response
	userWithLastTrack, err := s.userRepo.GetUserWithLastTrack(ctx, user.ID)
	if err != nil {
		s.logger.Error("Failed to get user with last track", "user_id", user.ID, "error", err)
		// Fallback to basic user if last track fetch fails
		userWithLastTrack = user
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)

	return &models.AuthResponse{
		Token: token,
		User:  *userWithLastTrack,
	}, nil
}

// GenerateToken generates a JWT token for a user
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	email := ""
	if user.Email != nil {
		email = *user.Email
	}
	claims := &Claims{
		UserID: user.ID,
		Email:  email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the user ID and role
func (s *AuthService) ValidateToken(tokenString string) (int, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return 0, "", fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, claims.Role, nil
	}

	return 0, "", errors.New("invalid token")
}

// GuestLogin creates a new guest user and returns a token
func (s *AuthService) GuestLogin(ctx context.Context) (*models.GuestResponse, error) {
	// Create guest user
	user := &models.User{
		Email:        nil, // NULL for guests
		PasswordHash: nil, // NULL for guests
		Provider:     nil, // NULL for guests
		ExternalID:   nil, // NULL for guests
		Role:         "guest",
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create guest user", "error", err)
		return nil, fmt.Errorf("failed to create guest user: %w", err)
	}

	// Update last login time
	now := time.Now()
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, now); err != nil {
		s.logger.Warn("Failed to update last login time", "user_id", user.ID, "error", err)
		// Continue anyway
	}

	// Generate token for guest
	token, err := s.GenerateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("Guest user created and logged in", "user_id", user.ID)

	return &models.GuestResponse{
		Token: token,
		User:  *user,
	}, nil
}

// PromoteGuestToUser promotes a guest user to a registered user via OAuth
func (s *AuthService) PromoteGuestToUser(ctx context.Context, guestID int, userInfo *models.OAuthUserInfo) (*models.AuthResponse, error) {
	// Update guest user with email and provider
	user, err := s.userRepo.GetUserByID(ctx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to find guest user: %w", err)
	}

	// Prepare optional fields
	var name, avatarURL *string
	if userInfo.Name != "" {
		name = &userInfo.Name
	}
	if userInfo.AvatarURL != "" {
		avatarURL = &userInfo.AvatarURL
	}

	// Update user fields to become a regular user
	updatedUser := &models.User{
		ID:         user.ID,
		Email:      &userInfo.Email,
		Name:       name,
		AvatarURL:  avatarURL,
		Provider:   &userInfo.Provider,
		ExternalID: &userInfo.ExternalID,
		Role:       "user", // Promote from guest to user
	}

	if err := s.userRepo.UpdateUser(ctx, updatedUser); err != nil {
		s.logger.Error("Failed to promote guest user", "guest_id", guestID, "error", err)
		return nil, fmt.Errorf("failed to promote guest: %w", err)
	}

	// Get updated user
	user, err = s.userRepo.GetUserByID(ctx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get promoted user: %w", err)
	}

	// Update last login time
	now := time.Now()
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, now); err != nil {
		s.logger.Warn("Failed to update last login time", "user_id", user.ID, "error", err)
	}

	// Generate new token for promoted user
	token, err := s.GenerateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("Guest user promoted to registered user", "guest_id", guestID, "email", userInfo.Email)

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}
