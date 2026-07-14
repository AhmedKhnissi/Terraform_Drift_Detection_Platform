package aws

// buildRegistry wires Terraform resource types to their AWS fetchers. Adding
// support for a new resource type is a matter of implementing a fetcher and
// registering it here — no changes to the provider core or comparator.
func buildRegistry(p *AWSProvider) map[string]ResourceFetcher {
	return map[string]ResourceFetcher{
		"aws_instance":       &ec2InstanceFetcher{client: p.ec2},
		"aws_vpc":            &ec2VPCFetcher{client: p.ec2},
		"aws_subnet":         &ec2SubnetFetcher{client: p.ec2},
		"aws_security_group": &ec2SGFetcher{client: p.ec2},
		"aws_s3_bucket":      &s3BucketFetcher{client: p.s3},
		"aws_db_instance":    &rdsInstanceFetcher{client: p.rds},
		"aws_iam_role":       &iamRoleFetcher{client: p.iam},
	}
}
