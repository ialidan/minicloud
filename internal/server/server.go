// Package server wires up the HTTP router, middleware, and handlers.
package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"minicloud/internal/config"
	"minicloud/internal/server/handler"
	"minicloud/internal/server/middleware"
	"minicloud/internal/service"
	"minicloud/web"
)

const maxHeaderSize = 1 << 20 // 1 MiB

// Server is the main HTTP server for minicloud.
type Server struct {
	cfg    *config.Config
	logger *slog.Logger
	http   *http.Server
	health *handler.Health
}

// New creates a Server with all routes and middleware configured.
func New(cfg *config.Config, logger *slog.Logger, authSvc *service.AuthService, fileSvc *service.FileService) *Server {
	r := chi.NewRouter()

	// Global middleware (order matters — see Iteration 1 notes).
	r.Use(middleware.RequestID)
	r.Use(middleware.SecureHeaders)
	r.Use(chimw.RealIP)
	r.Use(middleware.AccessLog(logger))
	r.Use(chimw.Recoverer)

	// Health probes — always available, no auth.
	health := handler.NewHealth()
	r.Get("/healthz", health.Liveness)
	r.Get("/readyz", health.Readiness)

	// Handlers.
	authHandler := handler.NewAuthHandler(authSvc, cfg.Server.SecureCookies)
	userHandler := handler.NewUserHandler(authSvc)
	fileHandler := handler.NewFileHandler(fileSvc, cfg.Server.MaxUploadSize)

	// Rate limiter for authentication endpoints: 5 failed attempts per
	// 15-minute window triggers a 15-minute lockout per IP.
	authRateLimit := middleware.NewRateLimiter(5, 15*time.Minute, 15*time.Minute)

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth endpoints.
		r.Get("/auth/setup", authHandler.CheckSetup)
		r.With(authRateLimit).Post("/auth/setup", authHandler.Setup)
		r.With(authRateLimit).Post("/auth/login", authHandler.Login)

		// Authenticated endpoints.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(authSvc))

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)

			// File operations.
			r.Post("/files", fileHandler.Upload)
			r.Get("/files", fileHandler.List)
			r.Get("/files/duplicates", fileHandler.ListDuplicates)
			r.Get("/files/{id}", fileHandler.Download)
			r.Put("/files/{id}/move", fileHandler.MoveFile)
			r.Delete("/files/{id}", fileHandler.Delete)

			// Directory operations.
			r.Get("/directories", fileHandler.ListAllDirectories)
			r.Post("/directories", fileHandler.CreateDirectory)
			r.Delete("/directories/{id}", fileHandler.DeleteDirectory)

			// Admin-only user management.
			r.Route("/admin/users", func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
				r.Patch("/{id}", userHandler.Update)
			})
		})
	})

	// Serve the embedded web UI — catch-all after API routes.
	staticFS, _ := fs.Sub(web.Static, "static")
	spa := handler.NewSPAHandler(staticFS)
	r.NotFound(spa.ServeHTTP)

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           r,
		ReadTimeout:       cfg.Server.ReadTimeout.Std(),
		WriteTimeout:      cfg.Server.WriteTimeout.Std(),
		IdleTimeout:       cfg.Server.IdleTimeout.Std(),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout.Std(),
		MaxHeaderBytes:    maxHeaderSize,
	}

	if cfg.Server.TLS.Enabled {
		srv.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return &Server{
		cfg:    cfg,
		logger: logger,
		http:   srv,
		health: health,
	}
}

// Health returns the health handler for registering readiness checkers.
func (s *Server) Health() *handler.Health {
	return s.health
}

// Run starts the HTTP(S) server and blocks until ctx is cancelled, then
// performs a graceful shutdown (drains in-flight requests up to 15 s).
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		var err error
		if s.cfg.Server.TLS.Enabled {
			s.logger.Info("starting TLS server", "addr", s.http.Addr)
			err = s.http.ListenAndServeTLS(
				s.cfg.Server.TLS.CertFile,
				s.cfg.Server.TLS.KeyFile,
			)
		} else {
			s.logger.Info("starting server", "addr", s.http.Addr)
			err = s.http.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.logger.Info("server stopped gracefully")
	return nil
}
