package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"

	"driftdetect/internal/cloud"
	"driftdetect/internal/model"
)

// rdsInstanceFetcher fetches aws_db_instance live state.
type rdsInstanceFetcher struct{ client *rds.Client }

func (f *rdsInstanceFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	out, err := f.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(exp.ID),
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}
	if len(out.DBInstances) == 0 {
		return model.ResourceState{}, cloud.ErrNotFound
	}
	db := out.DBInstances[0]
	var tags map[string]string
	if db.DBInstanceArn != nil {
		if tOut, tErr := f.client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
			ResourceName: db.DBInstanceArn,
		}); tErr == nil {
			tags = tagsToMap(tOut.TagList)
		}
	}
	return model.ResourceState{
		Provider: "aws",
		Type:     exp.Type,
		Name:     exp.Name,
		ID:       exp.ID,
		Attributes: map[string]interface{}{
			"engine":           aws.ToString(db.Engine),
			"db_instance_class": aws.ToString(db.DBInstanceClass),
			"allocated_storage": aws.ToInt32(db.AllocatedStorage),
			"multi_az":         aws.ToBool(db.MultiAZ),
		},
		Tags: tags,
	}, nil
}
