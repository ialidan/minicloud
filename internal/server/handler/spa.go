package handler

import (
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler serves embedded static files with SPA fallback.
// Any request that doesn't match a real file (and isn't an API route)
// gets index.html so client-side routing works.
type SPAHandler struct {
	fileServer http.Handler
	fs         fs.FS
}

// NewSPAHandler creates a handler that serves files from the given fs.FS.
// The fsys should be the "static" sub-directory of the embedded filesystem.
func NewSPAHandler(fsys fs.FS) *SPAHandler {
	return &SPAHandler{
		fileServer: http.FileServer(http.FS(fsys)),
		fs:         fsys,
	}
}

// ServeHTTP serves static files or falls back to index.html for SPA routes.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip leading slash for fs.Open.
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Check if the file exists in the embedded FS.
	if _, err := fs.Stat(h.fs, path); err != nil {
		// File doesn't exist — serve index.html (SPA fallback).
		r.URL.Path = "/"
	}

	h.fileServer.ServeHTTP(w, r)
}
