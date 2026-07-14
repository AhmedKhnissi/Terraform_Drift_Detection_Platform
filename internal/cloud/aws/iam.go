package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"driftdetect/internal/cloud"
	"driftdetect/internal/model"
)

// iamRoleFetcher fetches aws_iam_role live state.
type iamRoleFetcher struct{ client *iam.Client }

func (f *iamRoleFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	out, err := f.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(exp.ID),
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}
	if out.Role == nil {
		return model.ResourceState{}, cloud.ErrNotFound
	}
	r := out.Role
	return model.ResourceState{
		Provider: "aws",
		Type:     exp.Type,
		Name:     exp.Name,
		ID:       exp.ID,
		Attributes: map[string]interface{}{
			"arn":         aws.ToString(r.Arn),
			"description": aws.ToString(r.Description),
			"path":        aws.ToString(r.Path),
		},
		Tags: tagsToMap(r.Tags),
	}, nil
}
