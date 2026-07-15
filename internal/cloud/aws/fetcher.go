package aws

import (
	"errors"
	"reflect"
	"strings"

	"github.com/aws/smithy-go"

	"driftdetect/internal/cloud"
)

// ResourceFetcher fetches the live state of one AWS resource type. Re-declared
// here for convenience so fetcher files need not import the parent package.
type ResourceFetcher = cloud.ResourceFetcher

// tagsToMap normalizes any AWS SDK tag slice into a flat map. The various
// service Tag types (EC2, S3, RDS, IAM) all expose Key and Value as *string
// fields; reflecting over them lets a single helper cover every service
// without importing each one's generated types.
func tagsToMap(in any) map[string]string {
	if in == nil {
		return nil
	}
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil
	}
	if v.Len() == 0 {
		return nil
	}
	m := make(map[string]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		key := fieldString(elem, "Key")
		if key == "" {
			continue
		}
		m[key] = fieldString(elem, "Value")
	}
	return m
}

// fieldString reads a *string field (by name) from a struct value, returning
// "" when the field is absent or nil.
func fieldString(v reflect.Value, name string) string {
	f := v.FieldByName(name)
	if !f.IsValid() {
		return ""
	}
	if f.Kind() == reflect.Ptr {
		if f.IsNil() {
			return ""
		}
		if s, ok := f.Elem().Interface().(string); ok {
			return s
		}
	}
	return ""
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
