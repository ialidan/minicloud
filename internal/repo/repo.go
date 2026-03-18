// Package repo defines repository interfaces for data persistence.
// Implementations (e.g. sqlite) live in sub-packages.
package repo

import (
	"context"

	"minicloud/internal/domain"
)

// Pagination controls LIMIT/OFFSET for list queries.
// A nil *Pagination means "return all rows" (no limit).
type Pagination struct {
	Limit  int
	Offset int
}

// UserRepo handles user persistence.
type UserRepo interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	List(ctx context.Context, page *Pagination) ([]domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Count(ctx context.Context) (int, error)
	CountByRoleActive(ctx context.Context, role string) (int, error)
}

// FileRepo handles file metadata persistence.
type FileRepo interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id string) (*domain.File, error)
	ListByOwner(ctx context.Context, ownerID string, virtualPath string, page *Pagination) ([]domain.File, error)
	ListByOwnerAndMimePrefixes(ctx context.Context, ownerID string, prefixes []string, page *Pagination) ([]domain.File, error)
	ListByOwnerAndMimePrefixesInPath(ctx context.Context, ownerID string, prefixes []string, virtualPath string, page *Pagination) ([]domain.File, error)
	SearchByOwner(ctx context.Context, ownerID string, query string, page *Pagination) ([]domain.File, error)
	FindDuplicates(ctx context.Context, ownerID string) ([]domain.File, error)
	UpdateVirtualPath(ctx context.Context, id string, newPath string) error
	Delete(ctx context.Context, id string) error
}

// DirectoryRepo handles directory metadata persistence.
type DirectoryRepo interface {
	Create(ctx context.Context, dir *domain.Directory) error
	GetByID(ctx context.Context, id string) (*domain.Directory, error)
	ListByOwner(ctx context.Context, ownerID string, parentPath string) ([]domain.Directory, error)
	ListAllByOwner(ctx context.Context, ownerID string) ([]domain.Directory, error)
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, ownerID string, fullPath string) (bool, error)
}

// SessionRepo handles session persistence.
type SessionRepo interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	DeleteByID(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}
