package storage

import (
	"path/filepath"
	"testing"
	"time"

	"driftdetect/internal/model"
)

func sampleReport() model.DriftReport {
	return model.DriftReport{
		ScanID:        "deadbeef",
		StartedAt:     time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC),
		DurationMs:    42,
		ResourceCount: 2,
		DriftCount:    1,
		Summary:       map[model.DriftType]int{model.DriftDeleted: 1},
		Items: []model.DriftItem{
			{
				Type:      "aws_instance",
				Name:      "web",
				ID:        "i-0abc123",
				DriftType: model.DriftDeleted,
				Message:   "resource exists in state but was not found in the cloud",
			},
		},
	}
}

func TestSQLiteRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	if err := store.SaveScan(sampleReport()); err != nil {
		t.Fatalf("save: %v", err)
	}

	scans, err := store.ListScans(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(scans) != 1 || scans[0].ScanID != "deadbeef" || scans[0].DriftCount != 1 {
		t.Fatalf("unexpected list: %+v", scans)
	}

	got, err := store.GetReport("deadbeef")
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if got.DriftCount != 1 || len(got.Items) != 1 {
		t.Fatalf("unexpected report: %+v", got)
	}
	if got.Items[0].ID != "i-0abc123" {
		t.Fatalf("unexpected item: %+v", got.Items[0])
	}

	if _, err := store.GetReport("missing"); err == nil {
		t.Fatal("expected error for missing scan")
	}
}
