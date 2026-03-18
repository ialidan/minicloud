// Package migrations holds embedded SQL migration files.
package migrations

import "embed"

// FS contains all .sql migration files, embedded at compile time.
//
//go:embed *.sql
var FS embed.FS
