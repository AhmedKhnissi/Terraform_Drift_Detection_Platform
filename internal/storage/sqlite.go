package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"

	"driftdetect/internal/model"
)

// SQLiteStore is a Store backed by a local SQLite database file (pure Go via
// modernc.org/sqlite, no cgo — works on Windows without a C toolchain).
type SQLiteStore struct {
	db *sql.DB
}

// schema is applied on open. scans holds run metadata; drift_items holds the
// per-difference rows for each scan.
const schema = `
CREATE TABLE IF NOT EXISTS scans (
  id            TEXT PRIMARY KEY,
  started_at    DATETIME NOT NULL,
  duration_ms   INTEGER NOT NULL,
  resource_count INTEGER NOT NULL,
  drift_count   INTEGER NOT NULL,
  summary_json  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS drift_items (
  scan_id   TEXT NOT NULL,
  type      TEXT NOT NULL,
  name      TEXT NOT NULL,
  id        TEXT NOT NULL,
  drift_type TEXT NOT NULL,
  attribute TEXT,
  expected_json TEXT,
  actual_json   TEXT,
  message   TEXT
);
CREATE INDEX IF NOT EXISTS idx_drift_items_scan ON drift_items(scan_id);
`

// OpenSQLite opens (creating if needed) the SQLite database at path.
func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	db.SetMaxOpenConns(1) // sqlite is single-writer; avoid "database is locked".
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init sqlite schema: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

// Close closes the database handle.
func (s *SQLiteStore) Close() error { return s.db.Close() }

// SaveScan persists a full DriftReport inside a transaction.
func (s *SQLiteStore) SaveScan(report model.DriftReport) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	summary, err := json.Marshal(report.Summary)
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	_, err = tx.Exec(
		`INSERT INTO scans (id, started_at, duration_ms, resource_count, drift_count, summary_json)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		report.ScanID, report.StartedAt, report.DurationMs,
		report.ResourceCount, report.DriftCount, string(summary),
	)
	if err != nil {
		return fmt.Errorf("insert scan: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO drift_items (scan_id, type, name, id, drift_type, attribute, expected_json, actual_json, message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, it := range report.Items {
		ev, _ := json.Marshal(it.Expected)
		av, _ := json.Marshal(it.Actual)
		if _, err := stmt.Exec(
			report.ScanID, it.Type, it.Name, it.ID,
			string(it.DriftType), nullString(it.Attribute),
			nullBytes(ev), nullBytes(av), nullString(it.Message),
		); err != nil {
			return fmt.Errorf("insert drift item: %w", err)
		}
	}
	return tx.Commit()
}

// ListScans returns the most recent scan summaries, newest first.
func (s *SQLiteStore) ListScans(limit int) ([]model.ScanSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, started_at, duration_ms, resource_count, drift_count
		 FROM scans ORDER BY started_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.ScanSummary
	for rows.Next() {
		var ss model.ScanSummary
		if err := rows.Scan(&ss.ScanID, &ss.StartedAt, &ss.DurationMs, &ss.ResourceCount, &ss.DriftCount); err != nil {
			return nil, err
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

// GetScan returns the summary for a single scan.
func (s *SQLiteStore) GetScan(id string) (model.ScanSummary, error) {
	var ss model.ScanSummary
	err := s.db.QueryRow(
		`SELECT id, started_at, duration_ms, resource_count, drift_count
		 FROM scans WHERE id = ?`, id,
	).Scan(&ss.ScanID, &ss.StartedAt, &ss.DurationMs, &ss.ResourceCount, &ss.DriftCount)
	if err == sql.ErrNoRows {
		return model.ScanSummary{}, fmt.Errorf("scan %q not found", id)
	}
	return ss, err
}

// GetReport returns the full DriftReport for a scan, including its items.
func (s *SQLiteStore) GetReport(id string) (model.DriftReport, error) {
	var report model.DriftReport
	var summaryJSON string
	err := s.db.QueryRow(
		`SELECT id, started_at, duration_ms, resource_count, drift_count, summary_json
		 FROM scans WHERE id = ?`, id,
	).Scan(&report.ScanID, &report.StartedAt, &report.DurationMs,
		&report.ResourceCount, &report.DriftCount, &summaryJSON)
	if err == sql.ErrNoRows {
		return model.DriftReport{}, fmt.Errorf("scan %q not found", id)
	}
	if err != nil {
		return model.DriftReport{}, err
	}
	if err := json.Unmarshal([]byte(summaryJSON), &report.Summary); err != nil {
		return model.DriftReport{}, fmt.Errorf("unmarshal summary: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT type, name, id, drift_type, attribute, expected_json, actual_json, message
		 FROM drift_items WHERE scan_id = ?`, id,
	)
	if err != nil {
		return model.DriftReport{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			dt, attr, expJSON, actJSON, msg sql.NullString
			it                                           model.DriftItem
		)
		if err := rows.Scan(&it.Type, &it.Name, &it.ID, &dt, &attr, &expJSON, &actJSON, &msg); err != nil {
			return model.DriftReport{}, err
		}
		it.DriftType = model.DriftType(dt.String)
		it.Attribute = attr.String
		it.Message = msg.String
		if expJSON.Valid && expJSON.String != "" {
			_ = json.Unmarshal([]byte(expJSON.String), &it.Expected)
		}
		if actJSON.Valid && actJSON.String != "" {
			_ = json.Unmarshal([]byte(actJSON.String), &it.Actual)
		}
		report.Items = append(report.Items, it)
	}
	return report, rows.Err()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
