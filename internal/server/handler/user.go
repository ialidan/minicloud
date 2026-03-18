package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"minicloud/internal/domain"
	"minicloud/internal/repo"
	"minicloud/internal/service"
)

// userService defines the auth operations needed by UserHandler.
type userService interface {
	ListUsers(ctx context.Context, page *repo.Pagination) ([]domain.User, error)
	CreateUser(ctx context.Context, username, password, email, role string) (*domain.User, error)
	UpdateUser(ctx context.Context, id string, updates service.UserUpdates) (*domain.User, error)
}

// UserHandler handles admin user management endpoints.
type UserHandler struct {
	authSvc userService
}

func NewUserHandler(authSvc userService) *UserHandler {
	return &UserHandler{authSvc: authSvc}
}

// List returns all users.
//
//	GET /api/v1/admin/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page := parsePagination(r, 50, 200)
	users, err := h.authSvc.ListUsers(r.Context(), page)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	resp := make([]userResponse, len(users))
	for i := range users {
		resp[i] = toUserResponse(&users[i])
	}

	respondJSON(w, http.StatusOK, map[string]any{"users": resp})
}

// Create adds a new user.
//
//	POST /api/v1/admin/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if msg := decodeJSON(w, r, &req); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := h.authSvc.CreateUser(r.Context(), req.Username, req.Password, req.Email, req.Role)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"user": toUserResponse(user)})
}

// Update modifies an existing user.
//
//	PATCH /api/v1/admin/users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "user ID required")
		return
	}

	var req struct {
		Email    *string `json:"email"`
		Role     *string `json:"role"`
		IsActive *bool   `json:"is_active"`
		Password *string `json:"password"`
	}
	if msg := decodeJSON(w, r, &req); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	user, err := h.authSvc.UpdateUser(r.Context(), id, service.UserUpdates{
		Email:    req.Email,
		Role:     req.Role,
		IsActive: req.IsActive,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"user": toUserResponse(user)})
}
