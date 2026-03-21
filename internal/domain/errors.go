package domain

import "errors"

var (
	// ErrNotFound indicates the requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists indicates a uniqueness constraint violation.
	ErrAlreadyExists = errors.New("already exists")

	// ErrUnauthorized indicates missing or invalid credentials.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the user lacks permission for the action.
	ErrForbidden = errors.New("forbidden")

	// ErrDirectoryNotEmpty indicates a directory cannot be deleted because it contains items.
	ErrDirectoryNotEmpty = errors.New("directory not empty")

	// ErrFileTooLarge indicates the uploaded file exceeds the maximum allowed size.
	ErrFileTooLarge = errors.New("file too large")

	// ErrUnknownCategory indicates the requested browse category is not recognized.
	ErrUnknownCategory = errors.New("unknown category")
)
