package domain

import "time"

// Directory represents a virtual directory in a user's file tree.
type Directory struct {
	ID         string
	OwnerID    string
	ParentPath string // e.g. "/" or "/docs/"
	Name       string // directory display name
	CreatedAt  time.Time
}
