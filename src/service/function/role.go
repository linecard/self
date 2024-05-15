package function

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	types "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
)

func (s Service) GetRole(ctx context.Context, name string) (*iam.GetRoleOutput, error) {
	getRoleInput := &iam.GetRoleInput{
		RoleName: aws.String(name),
	}
	return s.Client.Iam.GetRole(ctx, getRoleInput)
}

func (s Service) GetRolePolicies(ctx context.Context, name string) (*iam.ListAttachedRolePoliciesOutput, error) {
	getRolePoliciesInput := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(name),
	}
	return s.Client.Iam.ListAttachedRolePolicies(ctx, getRolePoliciesInput)
}

func (s Service) DeleteRole(ctx context.Context, name string) (*iam.DeleteRoleOutput, error) {
	deleteRoleInput := &iam.DeleteRoleInput{
		RoleName: aws.String(name),
	}
	return s.Client.Iam.DeleteRole(ctx, deleteRoleInput)
}

func (s Service) PutRole(ctx context.Context, name string, document string, tags map[string]string) (*iam.GetRoleOutput, error) {
	var apiErr smithy.APIError
	var err error

	var iamTags []types.Tag
	for key, value := range tags {
		iamTags = append(iamTags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	createRoleInput := iam.CreateRoleInput{
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(document),
		Tags:                     iamTags,
	}

	updateAssumeRolePolicyInput := iam.UpdateAssumeRolePolicyInput{
		RoleName:       aws.String(name),
		PolicyDocument: aws.String(document),
	}

	_, err = s.Client.Iam.CreateRole(ctx, &createRoleInput)
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "EntityAlreadyExists":
			if _, err := s.Client.Iam.UpdateAssumeRolePolicy(ctx, &updateAssumeRolePolicyInput); err != nil {
				return &iam.GetRoleOutput{}, err
			}

			if _, err := s.updateRoleTags(ctx, name, tags); err != nil {
				return &iam.GetRoleOutput{}, err
			}
		default:
			return &iam.GetRoleOutput{}, err
		}
	}

	getRoleInput := iam.GetRoleInput{
		RoleName: aws.String(name),
	}
	waiter := iam.GetRoleAPIClient(s.Client.Iam)
	return waiter.GetRole(ctx, &getRoleInput)
}

func (s Service) AttachPolicyToRole(ctx context.Context, policyArn, roleName string) (*iam.AttachRolePolicyOutput, error) {
	attachRolePolicyInput := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String(policyArn),
		RoleName:  aws.String(roleName),
	}
	return s.Client.Iam.AttachRolePolicy(ctx, attachRolePolicyInput)
}

func (s Service) DetachPolicyFromRole(ctx context.Context, policyArn, roleName string) (*iam.DetachRolePolicyOutput, error) {
	detachRolePolicyInput := &iam.DetachRolePolicyInput{
		PolicyArn: aws.String(policyArn),
		RoleName:  aws.String(roleName),
	}
	return s.Client.Iam.DetachRolePolicy(ctx, detachRolePolicyInput)
}

func (s Service) updateRoleTags(ctx context.Context, name string, tags map[string]string) (*iam.ListRoleTagsOutput, error) {
	var tagKeys []string
	var iamTags []types.Tag

	listRoleTagsInput := &iam.ListRoleTagsInput{
		RoleName: aws.String(name),
	}

	listRoleTagsOutput, err := s.Client.Iam.ListRoleTags(ctx, listRoleTagsInput)
	if err != nil {
		return &iam.ListRoleTagsOutput{}, err
	}

	for _, tag := range listRoleTagsOutput.Tags {
		tagKeys = append(tagKeys, *tag.Key)
	}

	if len(tagKeys) > 0 {
		untagRoleInput := iam.UntagRoleInput{
			RoleName: aws.String(name),
			TagKeys:  tagKeys,
		}

		_, err = s.Client.Iam.UntagRole(ctx, &untagRoleInput)

		if err != nil {
			return &iam.ListRoleTagsOutput{}, err
		}
	}

	for key, value := range tags {
		iamTags = append(iamTags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	if len(iamTags) > 0 {
		tagRoleInput := iam.TagRoleInput{
			RoleName: aws.String(name),
			Tags:     iamTags,
		}

		_, err = s.Client.Iam.TagRole(ctx, &tagRoleInput)

		if err != nil {
			return &iam.ListRoleTagsOutput{}, err
		}
	}

	listRoleTagsOutput, err = s.Client.Iam.ListRoleTags(ctx, listRoleTagsInput)
	if err != nil {
		return &iam.ListRoleTagsOutput{}, err
	}

	return listRoleTagsOutput, nil
}
