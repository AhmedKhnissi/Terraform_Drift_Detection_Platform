// Package aws implements the cloud.CloudProvider contract for Amazon Web
// Services. It maps Terraform resource types to AWS API calls via a pluggable
// registry of ResourceFetcher implementations, so adding support for a new
// resource type only requires a new fetcher + one registry entry.
package aws

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"driftdetect/internal/cloud"
	appconfig "driftdetect/internal/config"
	"driftdetect/internal/model"
)

// AWSProvider fetches live AWS resource state.
type AWSProvider struct {
	cfg      aws.Config
	ec2      *ec2.Client
	s3       *s3.Client
	rds      *rds.Client
	iam      *iam.Client
	registry map[string]ResourceFetcher
}

// New builds an AWSProvider from an already-resolved AWS configuration.
func New(cfg aws.Config) *AWSProvider {
	p := &AWSProvider{
		cfg: cfg,
		ec2: ec2.NewFromConfig(cfg),
		s3:  s3.NewFromConfig(cfg),
		rds: rds.NewFromConfig(cfg),
		iam: iam.NewFromConfig(cfg),
	}
	p.registry = buildRegistry(p)
	return p
}

// LoadConfig resolves an aws.Config from the platform configuration, honoring
// region, shared profile, and optional inline static credentials.
func LoadConfig(ctx context.Context, c appconfig.AWSConfig) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(c.Region),
	}
	if c.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(c.Profile))
	}
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""),
		))
	}
	return config.LoadDefaultConfig(ctx, opts...)
}

// Name returns the provider identifier.
func (p *AWSProvider) Name() string { return "aws" }

// Supports reports whether this provider has a registered fetcher for the
// resource's type, so the engine can ignore unsupported resource types rather
// than reporting them as drift.
func (p *AWSProvider) Supports(rs model.ResourceState) bool {
	if rs.Provider != "aws" {
		return false
	}
	_, ok := p.registry[rs.Type]
	return ok
}

// Fetch returns the live state of every declared AWS resource this provider
// understands. Unknown resource types are skipped; resources that no longer
// exist in AWS surface as drift (deletion) via the comparator, not as errors.
func (p *AWSProvider) Fetch(ctx context.Context, expected []model.ResourceState) ([]model.ResourceState, error) {
	var actual []model.ResourceState
	for _, exp := range expected {
		if exp.Provider != "aws" {
			continue
		}
		f, ok := p.registry[exp.Type]
		if !ok {
			log.Printf("[warn] no AWS fetcher registered for type %s (skipping)", exp.Type)
			continue
		}
		got, err := f.Fetch(ctx, exp)
		if err != nil {
			if errors.Is(err, cloud.ErrNotFound) {
				// Resource gone from the cloud -> comparator reports deletion.
				continue
			}
			log.Printf("[warn] fetch %s %q failed: %v (skipping)", exp.Type, exp.Name, err)
			continue
		}
		actual = append(actual, got)
	}
	return actual, nil
}
