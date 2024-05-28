package registry

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

type EcrClient interface {
	BatchGetImage(ctx context.Context, params *ecr.BatchGetImageInput, optFns ...func(*ecr.Options)) (*ecr.BatchGetImageOutput, error)
	GetDownloadUrlForLayer(ctx context.Context, params *ecr.GetDownloadUrlForLayerInput, optFns ...func(*ecr.Options)) (*ecr.GetDownloadUrlForLayerOutput, error)
	DescribeImages(ctx context.Context, params *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeImagesOutput, error)
	GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error)
	BatchDeleteImage(ctx context.Context, params *ecr.BatchDeleteImageInput, optFns ...func(*ecr.Options)) (*ecr.BatchDeleteImageOutput, error)
	DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
	CreateRepository(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
}

type Client struct {
	Ecr EcrClient
}

type Service struct {
	Client Client
}

func FromClients(ecrClient EcrClient) Service {
	return Service{
		Client: Client{
			Ecr: ecrClient,
		},
	}
}
