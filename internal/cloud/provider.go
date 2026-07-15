// Package cloud defines the provider-agnostic contracts that let the drift
// engine compare Terraform state against any cloud. Concrete providers (AWS,
// and later Azure/GCP) implement these interfaces.
package cloud

import (
	"context"
	"errors"

	"driftdetect/internal/model"
)

// ErrNotFound is returned by a ResourceFetcher when the expected resource does
// not exist in the cloud. The provider treats this as drift (deletion) rather
// than a hard error.
var ErrNotFound = errors.New("resource not found in cloud")

// ResourceFetcher fetches the live state of a single declared resource.
// Implementations are keyed by Terraform resource type (e.g. "aws_instance").
type ResourceFetcher interface {
	// Fetch returns the actual ResourceState for the given expected resource.
	// If the resource no longer exists, return ErrNotFound.
	Fetch(ctx context.Context, expected model.ResourceState) (model.ResourceState, error)
}

// CloudProvider fetches the live state of all declared resources for a cloud.
type CloudProvider interface {
	// Name returns the provider identifier, e.g. "aws".
	Name() string
	// Fetch returns the actual ResourceState for every declared resource that
	// this provider knows how to inspect. Resources of unknown types are
	// skipped (not returned).
	Fetch(ctx context.Context, expected []model.ResourceState) ([]model.ResourceState, error)
	// Supports reports whether this provider can inspect the given declared
	// resource (i.e. it has a fetcher registered for its type). The engine uses
	// this to ignore resource types the provider cannot inspect, so they are
	// not falsely reported as drift.
	Supports(rs model.ResourceState) bool
}
