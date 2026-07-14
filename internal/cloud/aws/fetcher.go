package aws

import (
	"errors"
	"strings"

	"github.com/aws/smithy-go"

	"driftdetect/internal/cloud"
)

// ResourceFetcher fetches the live state of one AWS resource type. Re-declared
// here for convenience so fetcher files need not import the parent package.
type ResourceFetcher = cloud.ResourceFetcher

// tagsToMap normalizes any AWS SDK *Tag slice into a flat map. All AWS service
// tag types expose GetKey()/GetValue(), so a single generic helper covers EC2,
// S3, RDS, and IAM.
func tagsToMap[T interface {
	GetKey() string
	GetValue() string
}](tags []T) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[t.GetKey()] = t.GetValue()
	}
	return m
}

// isNotFound reports whether an AWS API error means the resource does not
// exist (so it should be reported as drift rather than a hard failure).
func isNotFound(err error) bool {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		code := ae.ErrorCode()
		return strings.Contains(code, "NotFound") ||
			strings.Contains(code, "NoSuch") ||
			strings.Contains(code, "DoesNotExist")
	}
	return false
}
