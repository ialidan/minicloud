// Package service contains business logic that sits between HTTP handlers
// and repository layer.
package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"minicloud/internal/auth"
	"minicloud/internal/domain"
	"minicloud/internal/repo"
)

const sessionDuration = 7 * 24 * time.Hour

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// AuthService handles authentication, session management, and user CRUD.
type AuthService struct {
	users    repo.UserRepo
	sessions repo.SessionRepo
	logger   *slog.Logger

	setupMu     sync.Mutex
	setupToken  string    // one-time admin bootstrap token; empty = setup done
	setupExpiry time.Time // when the setup token expires
}

// NewAuthService creates an AuthService with the given repositories.
func NewAuthService(users repo.UserRepo, sessions repo.SessionRepo, logger *slog.Logger) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
		logger:   logger,
	}
}

// ---------------------------------------------------------------------------
// Admin bootstrap
// ---------------------------------------------------------------------------

// InitSetup checks whether any users exist. If none, it generates and returns
// a one-time setup token. If users already exist, returns "".
func (s *AuthService) InitSetup(ctx context.Context) (string, error) {
	count, err := s.users.Count(ctx)
	if err != nil {
		return "", fmt.Errorf("counting users: %w", err)
	}
	if count > 0 {
		return "", nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating setup token: %w", err)
	}

	s.setupToken = hex.EncodeToString(b)
	s.setupExpiry = time.Now().Add(1 * time.Hour)
	return s.setupToken, nil
}

// NeedsSetup returns true if no admin has been created yet.
func (s *AuthService) NeedsSetup() bool {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if s.setupToken != "" && time.Now().After(s.setupExpiry) {
		s.setupToken = ""
		return false
	}
	return s.setupToken != ""
}

// Setup creates the initial admin user. The one-time token is invalidated on
// success. Uses constant-time comparison for the token.
func (s *AuthService) Setup(ctx context.Context, token, username, password string) (*domain.User, error) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if s.setupToken == "" {
		return nil, fmt.Errorf("setup already completed: %w", domain.ErrForbidden)
	}

	if time.Now().After(s.setupExpiry) {
		s.setupToken = "" // invalidate
		return nil, fmt.Errorf("setup token has expired, restart the server to generate a new one: %w", domain.ErrForbidden)
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(s.setupToken)) != 1 {
		return nil, fmt.Errorf("invalid setup token: %w", domain.ErrUnauthorized)
	}

	user, err := s.createUser(ctx, username, password, "", domain.RoleAdmin)
	if err != nil {
		return nil, err
	}

	s.setupToken = "" // one-time use
	s.logger.Info("initial admin user created", "username", username)
	return user, nil
}

// ---------------------------------------------------------------------------
// Authentication
// ---------------------------------------------------------------------------

// Login authenticates a user and creates a session. Returns ErrUnauthorized
// for any credential issue (wrong username, wrong password, inactive user)
// to avoid leaking which part failed.
func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.Session, *domain.User, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrUnauthorized
		}
		return nil, nil, fmt.Errorf("looking up user: %w", err)
	}

	if !user.IsActive {
		return nil, nil, domain.ErrUnauthorized
	}

	ok, err := auth.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return nil, nil, domain.ErrUnauthorized
	}

	// Enforce single active session per user. Old sessions are invalidated
	// when a new login occurs, preventing session accumulation.
	if err := s.sessions.DeleteByUserID(ctx, user.ID); err != nil {
		return nil, nil, fmt.Errorf("clearing old sessions: %w", err)
	}

	now := time.Now().UTC()
	session := &domain.Session{
		ID:        domain.NewID(),
		UserID:    user.ID,
		ExpiresAt: now.Add(sessionDuration),
		CreatedAt: now,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("creating session: %w", err)
	}

	s.logger.Info("user logged in", "username", username, "session_id", session.ID)
	return session, user, nil
}

// Logout deletes a session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.DeleteByID(ctx, sessionID)
}

// ValidateSession checks if a session is valid and returns the user.
// Implements middleware.SessionValidator.
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*domain.User, *domain.Session, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, domain.ErrUnauthorized
	}

	return user, session, nil
}

// ---------------------------------------------------------------------------
// User CRUD (admin operations)
// ---------------------------------------------------------------------------

// CreateUser creates a new user with the given credentials and role.
func (s *AuthService) CreateUser(ctx context.Context, username, password, email, role string) (*domain.User, error) {
	return s.createUser(ctx, username, password, email, role)
}

// ListUsers returns all users, with optional pagination.
func (s *AuthService) ListUsers(ctx context.Context, page *repo.Pagination) ([]domain.User, error) {
	return s.users.List(ctx, page)
}

// GetUser returns a user by ID.
func (s *AuthService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// UpdateUser applies partial updates to a user.
func (s *AuthService) UpdateUser(ctx context.Context, id string, updates UserUpdates) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if updates.Email != nil {
		user.Email = *updates.Email
	}
	if updates.Role != nil {
		r := *updates.Role
		if r != domain.RoleAdmin && r != domain.RoleUser {
			return nil, fmt.Errorf("role must be %q or %q", domain.RoleAdmin, domain.RoleUser)
		}
		user.Role = r
	}
	if updates.IsActive != nil {
		if !*updates.IsActive && user.Role == domain.RoleAdmin {
			count, err := s.users.CountByRoleActive(ctx, domain.RoleAdmin)
			if err != nil {
				return nil, fmt.Errorf("counting active admins: %w", err)
			}
			if count <= 1 {
				return nil, fmt.Errorf("cannot deactivate the last admin")
			}
		}
		user.IsActive = *updates.IsActive
	}
	if updates.Password != nil {
		if len(*updates.Password) < 8 {
			return nil, fmt.Errorf("password must be at least 8 characters")
		}
		hash, err := auth.HashPassword(*updates.Password)
		if err != nil {
			return nil, fmt.Errorf("hashing password: %w", err)
		}
		user.PasswordHash = hash
	}

	user.UpdatedAt = time.Now().UTC()
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UserUpdates holds optional fields for partial user updates.
// Nil pointers mean "don't change".
type UserUpdates struct {
	Email    *string
	Role     *string
	IsActive *bool
	Password *string
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (s *AuthService) createUser(ctx context.Context, username, password, email, role string) (*domain.User, error) {
	if !validUsername.MatchString(username) {
		return nil, fmt.Errorf("username must be 3-32 alphanumeric characters or underscores")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}
	if role != domain.RoleAdmin && role != domain.RoleUser {
		role = domain.RoleUser
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           domain.NewID(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
