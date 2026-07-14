// Package web serves the simple drift dashboard (HTML) and a JSON API for
// programmatic access. It is intentionally dependency-free beyond the standard
// library, using html/template with embedded assets.
package web

import (
	"context"
	"embed"
	"html/template"
	"net"
	"net/http"

	"driftdetect/internal/model"
	"driftdetect/internal/storage"
)

//go:embed templates/*.html
var templateFS embed.FS

// Server hosts the dashboard and API. scanFn, if set, lets the dashboard trigger
// on-demand scans; it may be nil when only viewing history is desired.
type Server struct {
	store  storage.Store
	scanFn func(ctx context.Context) (model.DriftReport, error)
	addr   string
	tmpl   *template.Template
}

// NewServer builds a dashboard server backed by the given store.
func NewServer(store storage.Store, scanFn func(ctx context.Context) (model.DriftReport, error), addr string) (*Server, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{store: store, scanFn: scanFn, addr: addr, tmpl: tmpl}, nil
}

// Handler returns the HTTP handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/scans/", s.handleScanDetail)
	mux.HandleFunc("/api/scans", s.handleAPIScans)
	mux.HandleFunc("/api/scans/", s.handleAPIScan)
	mux.HandleFunc("/api/scan", s.handleAPITrigger) // POST trigger
	return mux
}

// Start begins serving (blocking). Pass a context to allow graceful shutdown.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{Addr: s.addr, Handler: s.Handler(), BaseContext: func(_ net.Listener) context.Context { return ctx }}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	}
}
