// Package state reads Terraform state from various backends and normalizes it
// into the cloud-agnostic model.ResourceState shape.
package state

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"

	"driftdetect/internal/model"
)

// StateSource provides raw Terraform state bytes from a backend.
type StateSource interface {
	// Load returns a reader for the state document. The caller must Close it.
	Load(ctx context.Context) (io.ReadCloser, error)
	// String describes the source for logging.
	String() string
}

// NewSource builds the appropriate StateSource for a raw source string.
// Local paths and file:// URLs use the local filesystem; s3:// URLs use an S3
// backend (requiring a configured aws.Config).
func NewSource(raw string, awsCfg aws.Config) (StateSource, error) {
	switch {
	case strings.HasPrefix(raw, "s3://"):
		return NewS3Source(raw, awsCfg)
	case strings.HasPrefix(raw, "file://"):
		return NewLocalSource(strings.TrimPrefix(raw, "file://")), nil
	default:
		return NewLocalSource(raw), nil
	}
}

// Parse reads a state document from a source and returns the normalized
// expected resources.
func Parse(ctx context.Context, src StateSource) ([]model.ResourceState, error) {
	rc, err := src.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	defer rc.Close()
	return ParseState(rc)
}
