// Package web holds the embedded static assets for the minicloud web UI.
// The built frontend is embedded into the Go binary via go:embed so that
// the final artifact is a single distributable file.
package web

import "embed"

// Static contains the web UI assets. In production this holds the compiled
// frontend; during early development it serves a placeholder page.
//
//go:embed static/*
var Static embed.FS
