package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"minicloud/internal/domain"
	"minicloud/internal/server/middleware"
)

const sessionCookieMaxAge = 7 * 24 * time.Hour

// authService defines the auth operations needed by AuthHandler.
type authService interface {
	NeedsSetup() bool
	Setup(ctx context.Context, token, username, password string) (*domain.User, error)
	Login(ctx context.Context, username, password string) (*domain.Session, *domain.User, error)
	Logout(ctx context.Context, sessionID string) error
	ValidateSession(ctx context.Context, sessionID string) (*domain.User, *domain.Session, error)
}

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authSvc      authService
	secureCookie bool
}

// NewAuthHandler creates an AuthHandler.
// secureCookie should be true when the app is behind HTTPS.
func NewAuthHandler(authSvc authService, secureCookie bool) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, secureCookie: secureCookie}
}

// CheckSetup returns whether the instance needs initial admin setup.
//
//	GET /api/v1/auth/setup
func (h *AuthHandler) CheckSetup(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{
		"needs_setup": h.authSvc.NeedsSetup(),
	})
}

// Setup creates the initial admin user.
//
//	POST /api/v1/auth/setup
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if msg := decodeJSON(w, r, &req); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	if req.Token == "" || req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "token, username, and password are required")
		return
	}

	user, err := h.authSvc.Setup(r.Context(), req.Token, req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrForbidden) {
			status = http.StatusForbidden
		} else if errors.Is(err, domain.ErrUnauthorized) {
			status = http.StatusUnauthorized
		} else if errors.Is(err, domain.ErrAlreadyExists) {
			status = http.StatusConflict
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"user": toUserResponse(user),
	})
}

// Login authenticates a user and sets a session cookie.
//
//	POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if msg := decodeJSON(w, r, &req); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	session, user, err := h.authSvc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionCookieMaxAge.Seconds()),
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"user": toUserResponse(user),
	})
}

// Logout destroys the current session and clears the cookie.
//
//	POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r.Context())
	if session != nil {
		h.authSvc.Logout(r.Context(), session.ID) //nolint:errcheck
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // delete
	})

	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// Me returns the current authenticated user.
//
//	GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"user": toUserResponse(user),
	})
}
