package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"driftdetect/internal/model"
)

// errNoTrigger is returned when on-demand scans are disabled on the server.
var errNoTrigger = errors.New("on-demand scan not configured on this server")

type indexData struct {
	Scans []model.ScanSummary
}

type reportData struct {
	Report model.DriftReport
}

// handleIndex renders the scan history dashboard.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	scans, err := s.store.ListScans(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "index.html", indexData{Scans: scans})
}

// handleScanDetail renders a single scan's drift report.
func (s *Server) handleScanDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/scans/")
	if id == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	report, err := s.store.GetReport(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.render(w, "report.html", reportData{Report: report})
}

// handleAPIScans returns the scan summary list as JSON.
func (s *Server) handleAPIScans(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/scans" {
		http.NotFound(w, r)
		return
	}
	scans, err := s.store.ListScans(200)
	if err != nil {
		s.writeJSONError(w, err, http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, scans)
}

// handleAPIScan returns a full drift report by id as JSON.
func (s *Server) handleAPIScan(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/scans/")
	if id == "" {
		http.Error(w, "missing scan id", http.StatusBadRequest)
		return
	}
	report, err := s.store.GetReport(id)
	if err != nil {
		s.writeJSONError(w, err, http.StatusNotFound)
		return
	}
	s.writeJSON(w, report)
}

// handleAPITrigger runs an on-demand scan (POST) and returns the report JSON.
func (s *Server) handleAPITrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.scanFn == nil {
		s.writeJSONError(w, errNoTrigger, http.StatusNotImplemented)
		return
	}
	report, err := s.scanFn(r.Context())
	if err != nil {
		s.writeJSONError(w, err, http.StatusInternalServerError)
		return
	}
	if err := s.store.SaveScan(report); err != nil {
		s.writeJSONError(w, err, http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, report)
}

func (s *Server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) writeJSONError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
