package domain

import "time"

// User represents a registered minicloud user.
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	Role         string // "admin" or "user"
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)
