package registry

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

func (s Service) List(ctx context.Context, registryUrl, repositoryName string) (ecr.DescribeImagesOutput, error) {
	registryId := strings.Split(registryUrl, ".")[0]

	input := &ecr.DescribeImagesInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repositoryName),
	}

	output, err := s.Client.Ecr.DescribeImages(ctx, input)
	if err != nil {
		return ecr.DescribeImagesOutput{}, err
	}

	return *output, nil
}
