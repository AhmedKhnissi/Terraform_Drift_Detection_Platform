package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"driftdetect/internal/cloud"
	"driftdetect/internal/model"
)

// ec2InstanceFetcher fetches aws_instance live state.
type ec2InstanceFetcher struct{ client *ec2.Client }

func (f *ec2InstanceFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	out, err := f.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{exp.ID},
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if aws.ToString(inst.InstanceId) != exp.ID {
				continue
			}
			return model.ResourceState{
				Provider: "aws",
				Type:     exp.Type,
				Name:     exp.Name,
				ID:       exp.ID,
				Attributes: map[string]interface{}{
					"instance_type": string(inst.InstanceType),
					"ami":           aws.ToString(inst.ImageId),
					"subnet_id":     aws.ToString(inst.SubnetId),
					"private_ip":    aws.ToString(inst.PrivateIpAddress),
					"public_ip":     aws.ToString(inst.PublicIpAddress),
				},
				Tags: tagsToMap(inst.Tags),
			}, nil
		}
	}
	return model.ResourceState{}, cloud.ErrNotFound
}

// ec2VPCFetcher fetches aws_vpc live state.
type ec2VPCFetcher struct{ client *ec2.Client }

func (f *ec2VPCFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	out, err := f.client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{exp.ID},
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}
	if len(out.Vpcs) == 0 {
		return model.ResourceState{}, cloud.ErrNotFound
	}
	v := out.Vpcs[0]
	return model.ResourceState{
		Provider: "aws",
		Type:     exp.Type,
		Name:     exp.Name,
		ID:       exp.ID,
		Attributes: map[string]interface{}{
			"cidr_block":        aws.ToString(v.CidrBlock),
			"instance_tenancy":  string(v.Tenancy),
			"enable_dns_support": aws.ToBool(v.EnableDnsSupport),
		},
		Tags: tagsToMap(v.Tags),
	}, nil
}

// ec2SubnetFetcher fetches aws_subnet live state.
type ec2SubnetFetcher struct{ client *ec2.Client }

func (f *ec2SubnetFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	out, err := f.client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{exp.ID},
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}
	if len(out.Subnets) == 0 {
		return model.ResourceState{}, cloud.ErrNotFound
	}
	s := out.Subnets[0]
	return model.ResourceState{
		Provider: "aws",
		Type:     exp.Type,
		Name:     exp.Name,
		ID:       exp.ID,
		Attributes: map[string]interface{}{
			"vpc_id":     aws.ToString(s.VpcId),
			"cidr_block": aws.ToString(s.CidrBlock),
			"available_ip_address_count": aws.ToInt32(s.AvailableIpAddressCount),
		},
		Tags: tagsToMap(s.Tags),
	}, nil
}

// ec2SGFetcher fetches aws_security_group live state.
type ec2SGFetcher struct{ client *ec2.Client }

func (f *ec2SGFetcher) Fetch(ctx context.Context, exp model.ResourceState) (model.ResourceState, error) {
	out, err := f.client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{exp.ID},
	})
	if err != nil {
		if isNotFound(err) {
			return model.ResourceState{}, cloud.ErrNotFound
		}
		return model.ResourceState{}, err
	}
	if len(out.SecurityGroups) == 0 {
		return model.ResourceState{}, cloud.ErrNotFound
	}
	g := out.SecurityGroups[0]
	return model.ResourceState{
		Provider: "aws",
		Type:     exp.Type,
		Name:     exp.Name,
		ID:       exp.ID,
		Attributes: map[string]interface{}{
			"vpc_id":      aws.ToString(g.VpcId),
			"description": aws.ToString(g.Description),
		},
		Tags: tagsToMap(g.Tags),
	}, nil
}
