// Package drift compares expected (Terraform state) and actual (cloud) resource
// states and produces a DriftReport describing the differences.
package drift

// driftAttributes lists the scalar attributes compared per resource type. Only
// these are diffed, which avoids false positives from computed/read-only fields
// that legitimately differ between state and the live API (e.g. private_ip,
// last_modified). To compare additional attributes, add them here.
var driftAttributes = map[string][]string{
	"aws_instance":       {"instance_type", "ami", "subnet_id"},
	"aws_vpc":            {"cidr_block", "instance_tenancy"},
	"aws_subnet":         {"vpc_id", "cidr_block"},
	"aws_security_group": {"vpc_id", "description"},
	"aws_s3_bucket":      {"versioning_status", "sse_algorithm"},
	"aws_db_instance":    {"engine", "db_instance_class", "allocated_storage", "multi_az"},
	"aws_iam_role":       {"description"},
}

// DriftAttributes returns the attribute keys compared for a given resource type.
func DriftAttributes(resourceType string) []string {
	return driftAttributes[resourceType]
}

// DriftOptions toggles which categories of drift are evaluated during a scan.
type DriftOptions struct {
	CompareAttributes bool
	CompareTags       bool
	DetectOrphans     bool
}
