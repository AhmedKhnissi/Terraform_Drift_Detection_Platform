package state

import (
	"context"
	"fmt"
	"io"
	"os"
)

// LocalSource reads Terraform state from the local filesystem.
type LocalSource struct {
	path string
}

// NewLocalSource creates a local filesystem state source.
func NewLocalSource(path string) *LocalSource {
	return &LocalSource{path: path}
}

// Load opens the local state file.
func (s *LocalSource) Load(_ context.Context) (io.ReadCloser, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open state file %q: %w", s.path, err)
	}
	return f, nil
}

// String returns a human-readable description of the source.
func (s *LocalSource) String() string {
	return "file://" + s.path
}
