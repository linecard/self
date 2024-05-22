package function

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
	"github.com/linecard/self/internal/util"
)

func (s Service) PutPolicy(ctx context.Context, arn string, document string, tags map[string]string) (*iam.GetPolicyOutput, error) {
	var apiErr smithy.APIError

	name := util.PolicyNameFromArn(arn)

	_, err := s.createPolicy(ctx, name, document, tags)
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "EntityAlreadyExists":
			break
		default:
			return &iam.GetPolicyOutput{}, err
		}
	}

	if _, err = s.updatePolicy(ctx, arn, document, tags); err != nil {
		return &iam.GetPolicyOutput{}, err
	}

	getPolicyInput := &iam.GetPolicyInput{
		PolicyArn: aws.String(arn),
	}
	return s.Client.Iam.GetPolicy(ctx, getPolicyInput)
}

func (s Service) DeletePolicy(ctx context.Context, arn string) (*iam.DeletePolicyOutput, error) {
	if _, err := s.garbageCollectPolicyVersions(ctx, arn); err != nil {
		return &iam.DeletePolicyOutput{}, err
	}

	deletePolicyInput := &iam.DeletePolicyInput{
		PolicyArn: aws.String(arn),
	}

	return s.Client.Iam.DeletePolicy(ctx, deletePolicyInput)
}

func (s Service) createPolicy(ctx context.Context, name, document string, tags map[string]string) (*iam.CreatePolicyOutput, error) {
	createPolicyInput := &iam.CreatePolicyInput{
		PolicyName:     aws.String(name),
		PolicyDocument: aws.String(document),
	}

	for key, value := range tags {
		createPolicyInput.Tags = append(createPolicyInput.Tags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	createPolicyOutput, err := s.Client.Iam.CreatePolicy(ctx, createPolicyInput)
	if err != nil {
		return &iam.CreatePolicyOutput{}, err
	}

	return createPolicyOutput, nil
}

func (s Service) updatePolicy(ctx context.Context, arn, document string, tags map[string]string) (*iam.CreatePolicyVersionOutput, error) {
	_, err := s.garbageCollectPolicyVersions(ctx, arn)
	if err != nil {
		return &iam.CreatePolicyVersionOutput{}, err
	}

	createPolicyVersionInput := &iam.CreatePolicyVersionInput{
		PolicyArn:      aws.String(arn),
		PolicyDocument: aws.String(document),
		SetAsDefault:   *aws.Bool(true),
	}

	createPolicyVersionOutput, err := s.Client.Iam.CreatePolicyVersion(ctx, createPolicyVersionInput)
	if err != nil {
		return &iam.CreatePolicyVersionOutput{}, err
	}

	_, err = s.updatePolicyTags(ctx, arn, tags)
	if err != nil {
		return createPolicyVersionOutput, err
	}

	return createPolicyVersionOutput, nil
}

func (s Service) garbageCollectPolicyVersions(ctx context.Context, arn string) ([]types.PolicyVersion, error) {
	var apiErr smithy.APIError
	var deleted []types.PolicyVersion

	listPolicyVersionsInput := &iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(arn),
	}

	listPolicyVersionsOutput, err := s.Client.Iam.ListPolicyVersions(ctx, listPolicyVersionsInput)
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchEntity" {
		return deleted, nil
	} else if err != nil {
		return deleted, err
	}

	for _, version := range listPolicyVersionsOutput.Versions {
		if !version.IsDefaultVersion {
			deletePolicyVersionInput := &iam.DeletePolicyVersionInput{
				PolicyArn: aws.String(arn),
				VersionId: version.VersionId,
			}

			_, err := s.Client.Iam.DeletePolicyVersion(ctx, deletePolicyVersionInput)
			if err != nil {
				return deleted, err
			}
			deleted = append(deleted, version)
		}
	}
	return deleted, nil
}

func (s Service) updatePolicyTags(ctx context.Context, arn string, tags map[string]string) (*iam.ListPolicyTagsOutput, error) {
	var removeTags []string
	var createTags []types.Tag

	listPolicyTagsInput := &iam.ListPolicyTagsInput{
		PolicyArn: aws.String(arn),
	}

	listPolicyTagsOutput, err := s.Client.Iam.ListPolicyTags(ctx, listPolicyTagsInput)
	if err != nil {
		return &iam.ListPolicyTagsOutput{}, err
	}

	for _, tag := range listPolicyTagsOutput.Tags {
		removeTags = append(removeTags, *tag.Key)
	}

	if len(removeTags) > 0 {
		untagPolicyInput := &iam.UntagPolicyInput{
			PolicyArn: aws.String(arn),
			TagKeys:   removeTags,
		}

		_, err = s.Client.Iam.UntagPolicy(ctx, untagPolicyInput)
		if err != nil {
			return &iam.ListPolicyTagsOutput{}, err
		}
	}

	for key, value := range tags {
		createTags = append(createTags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	if len(createTags) > 0 {
		_, err = s.Client.Iam.TagPolicy(ctx, &iam.TagPolicyInput{
			PolicyArn: aws.String(arn),
			Tags:      createTags,
		})
		if err != nil {
			return &iam.ListPolicyTagsOutput{}, err
		}
	}

	return s.Client.Iam.ListPolicyTags(ctx, listPolicyTagsInput)
}
