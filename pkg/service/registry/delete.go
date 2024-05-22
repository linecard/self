package registry

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func (s Service) Delete(ctx context.Context, registryId, repository string, imageDigests []string) error {
	batchDeleteImageInput := ecr.BatchDeleteImageInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repository),
		ImageIds:       []types.ImageIdentifier{},
	}

	for _, digest := range imageDigests {
		batchDeleteImageInput.ImageIds = append(batchDeleteImageInput.ImageIds, types.ImageIdentifier{
			ImageDigest: aws.String(digest),
		})
	}

	_, err := s.Client.Ecr.BatchDeleteImage(ctx, &batchDeleteImageInput)
	if err != nil {
		return err
	}

	return nil
}

func (s Service) Untag(ctx context.Context, registryId, repository, tag string) error {
	deleteInput := ecr.BatchDeleteImageInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repository),
		ImageIds: []types.ImageIdentifier{
			{
				ImageTag: aws.String(tag),
			},
		},
	}

	if _, err := s.Client.Ecr.BatchDeleteImage(ctx, &deleteInput); err != nil {
		return err
	}

	return nil
}
