package registry

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/smithy-go"
)

func (s Service) PutRepository(ctx context.Context, repositoryName string) error {
	var apiErr smithy.APIError

	_, err := s.Client.Ecr.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repositoryName},
	})

	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "RepositoryNotFoundException":
			_, err = s.Client.Ecr.CreateRepository(ctx, &ecr.CreateRepositoryInput{
				RepositoryName: aws.String(repositoryName),
			})

			if err != nil {
				return err
			}

		case "RepositoryAlreadyExistsException":
			return nil

		default:
			return err
		}
	}

	return nil
}
