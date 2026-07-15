// Package engine orchestrates a full drift scan: read state, fetch live cloud
// state, compare, and assemble a DriftReport with timing and a stable scan id.
package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"driftdetect/internal/cloud"
	"driftdetect/internal/drift"
	"driftdetect/internal/model"
	"driftdetect/internal/state"
)

// Runner performs a complete drift scan and returns the report.
type Runner struct {
	Source   state.StateSource
	Provider cloud.CloudProvider
	Options  drift.DriftOptions
}

// Run executes the scan. It reads Terraform state through the configured
// source, fetches the live cloud state, and diffs the two.
func (r *Runner) Run(ctx context.Context) (model.DriftReport, error) {
	start := time.Now()

	expected, err := state.Parse(ctx, r.Source)
	if err != nil {
		return model.DriftReport{}, err
	}

	// Only compare resources this provider can actually inspect. Resource types
	// without a registered fetcher (e.g. aws_s3_bucket_versioning) are ignored
	// rather than falsely reported as deleted.
	inspectable := make([]model.ResourceState, 0, len(expected))
	for _, e := range expected {
		if r.Provider.Supports(e) {
			inspectable = append(inspectable, e)
		}
	}

	actual, err := r.Provider.Fetch(ctx, expected)
	if err != nil {
		return model.DriftReport{}, err
	}

	report := drift.Compare(inspectable, actual, r.Options)
	report.ScanID = newScanID()
	report.StartedAt = start
	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

// newScanID returns a short random hex identifier for a scan run.
func newScanID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; fall back to a timestamp-derived id.
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000")))
	}
	return hex.EncodeToString(b[:])
}
