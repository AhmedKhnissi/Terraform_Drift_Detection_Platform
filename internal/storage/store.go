// Package storage persists scan runs and their drift items so they can be
// listed and reviewed later (history, dashboard, scheduled scans).
package storage

import "driftdetect/internal/model"

// Store is the persistence contract for drift scan results.
type Store interface {
	// SaveScan persists a complete DriftReport (summary + items).
	SaveScan(report model.DriftReport) error
	// ListScans returns the most recent scan summaries, newest first.
	ListScans(limit int) ([]model.ScanSummary, error)
	// GetScan returns the lightweight summary for a single scan.
	GetScan(id string) (model.ScanSummary, error)
	// GetReport returns the full DriftReport (including items) for a scan.
	GetReport(id string) (model.DriftReport, error)
	// Close releases the underlying database handle.
	Close() error
}
