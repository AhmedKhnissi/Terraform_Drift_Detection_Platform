// Package model defines the cloud-agnostic resource representation used
// throughout the drift detection platform, plus the report structures that
// describe configuration drift between declared (Terraform state) and actual
// (cloud provider API) infrastructure.
package model

import "time"

// ResourceState is the normalized representation of a single infrastructure
// resource. Both Terraform state and live cloud APIs are mapped into this shape
// so that they can be compared without caring about the underlying source.
type ResourceState struct {
	// Provider is the cloud provider, e.g. "aws".
	Provider string `json:"provider"`
	// Type is the Terraform resource type, e.g. "aws_instance".
	Type string `json:"type"`
	// Name is the Terraform logical name of the resource.
	Name string `json:"name"`
	// ID is the cloud identifier (instance id, bucket name, vpc id, ...) used to
	// match expected and actual resources of the same type.
	ID string `json:"id"`
	// Attributes holds drift-relevant scalar attributes (instance_type, ami,
	// cidr_block, ...). Computed/read-only fields are intentionally excluded by
	// the provider fetchers and the comparison rules.
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	// Tags holds the resource tags as a flat key/value map.
	Tags map[string]string `json:"tags,omitempty"`
}

// DriftType enumerates the kinds of drift a comparator can report.
type DriftType string

const (
	// DriftNone means the resource matches exactly.
	DriftNone DriftType = "none"
	// DriftDeleted means the resource exists in state but is gone from the cloud.
	DriftDeleted DriftType = "deleted"
	// DriftModified means a compared attribute changed value.
	DriftModified DriftType = "modified"
	// DriftTagChange means one or more tags were added, removed, or changed.
	DriftTagChange DriftType = "tag_change"
	// DriftOrphaned means the resource exists in the cloud but not in state.
	DriftOrphaned DriftType = "orphaned"
)

// AllDriftTypes lists every drift type, used for summary initialization.
var AllDriftTypes = []DriftType{
	DriftDeleted,
	DriftModified,
	DriftTagChange,
	DriftOrphaned,
}

// DriftItem is a single difference detected between expected and actual state.
type DriftItem struct {
	// Type / Name / ID identify the affected resource.
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id"`
	// DriftType classifies the difference.
	DriftType DriftType `json:"drift_type"`
	// Attribute is the changed attribute name (for DriftModified).
	Attribute string `json:"attribute,omitempty"`
	// Expected is the value declared in Terraform state.
	Expected interface{} `json:"expected,omitempty"`
	// Actual is the value observed in the cloud.
	Actual interface{} `json:"actual,omitempty"`
	// Message is a human-readable description of the difference.
	Message string `json:"message,omitempty"`
}

// DriftReport is the result of a single scan, containing a summary and the
// full list of detected drift items.
type DriftReport struct {
	ScanID        string         `json:"scan_id"`
	StartedAt     time.Time      `json:"started_at"`
	DurationMs    int64          `json:"duration_ms"`
	ResourceCount int            `json:"resource_count"`
	DriftCount    int            `json:"drift_count"`
	Items         []DriftItem    `json:"items"`
	Summary       map[string]int `json:"summary"`
}

// ScanSummary is a lightweight view of a persisted scan, used for listing.
type ScanSummary struct {
	ScanID        string    `json:"scan_id"`
	StartedAt     time.Time `json:"started_at"`
	DurationMs    int64     `json:"duration_ms"`
	ResourceCount int       `json:"resource_count"`
	DriftCount    int       `json:"drift_count"`
}
