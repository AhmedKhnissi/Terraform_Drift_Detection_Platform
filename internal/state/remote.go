package state

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Source reads Terraform state from an S3 backend (s3://bucket/key).
type S3Source struct {
	bucket string
	key    string
	client *s3.Client
}

// NewS3Source parses an s3://bucket/key URL and builds an S3 client from the
// provided AWS configuration.
func NewS3Source(raw string, cfg aws.Config) (*S3Source, error) {
	trimmed := strings.TrimPrefix(raw, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid s3 state source %q (expected s3://bucket/key)", raw)
	}
	return &S3Source{
		bucket: parts[0],
		key:    parts[1],
		client: s3.NewFromConfig(cfg),
	}, nil
}

// Load fetches the state object from S3.
func (s *S3Source) Load(ctx context.Context) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
	})
	if err != nil {
		return nil, fmt.Errorf("get state from s3://%s/%s: %w", s.bucket, s.key, err)
	}
	return out.Body, nil
}

// String returns a human-readable description of the source.
func (s *S3Source) String() string {
	return fmt.Sprintf("s3://%s/%s", s.bucket, s.key)
}
