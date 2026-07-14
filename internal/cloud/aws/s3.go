package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"driftdetect/internal/cloud"
	"driftdetect/internal/model"
)

// s3BucketFetcher fetches aws_s3_bucket live state (tags, versioning,
// encryption). Bucket existence is inferred from the tagging/versioning calls.
type s3BucketFetcher struct{ client *s3.Client }

func (f *s3BucketFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	bucket := exp.ID

	tagOut, err := f.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}

	attrs := map[string]interface{}{
		"bucket": bucket,
	}

	if vOut, err := f.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	}); err == nil {
		attrs["versioning_status"] = string(vOut.Status)
	}

	if eOut, err := f.client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucket),
	}); err == nil && eOut.ServerSideEncryptionConfiguration != nil {
		if len(eOut.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := eOut.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				attrs["sse_algorithm"] = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
			}
		}
	}

	return model.ResourceState{
		Provider:  "aws",
		Type:      exp.Type,
		Name:      exp.Name,
		ID:        bucket,
		Attributes: attrs,
		Tags:      tagsToMap(tagOut.TagSet),
	}, nil
}
