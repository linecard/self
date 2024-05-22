package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	dockerTypes "github.com/docker/docker/api/types"
)

type DistributionManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

func (s Service) InspectByTag(ctx context.Context, registryId, repository, tag string) (dockerTypes.ImageInspect, error) {
	batchGetImageInput := &ecr.BatchGetImageInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repository),
		ImageIds: []ecrTypes.ImageIdentifier{
			{
				ImageTag: aws.String(tag),
			},
		},
	}

	batchGetImageOutput, err := s.Client.Ecr.BatchGetImage(ctx, batchGetImageInput)
	if err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	if len(batchGetImageOutput.Images) > 1 {
		return dockerTypes.ImageInspect{}, fmt.Errorf("multiple releases found for tag %s", tag)
	}

	if len(batchGetImageOutput.Images) == 0 {
		return dockerTypes.ImageInspect{}, fmt.Errorf("no such release found for tag %s", tag)
	}

	return s.inspect(ctx, registryId, repository, batchGetImageOutput)
}

func (s Service) InspectByDigest(ctx context.Context, registryId, repository, digest string) (dockerTypes.ImageInspect, error) {
	batchGetImageInput := &ecr.BatchGetImageInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repository),
		ImageIds: []ecrTypes.ImageIdentifier{
			{
				ImageDigest: aws.String("sha256:" + digest),
			},
		},
	}

	batchGetImageOutput, err := s.Client.Ecr.BatchGetImage(ctx, batchGetImageInput)
	if err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	if len(batchGetImageOutput.Images) == 0 {
		return dockerTypes.ImageInspect{}, fmt.Errorf("no such release found for digest %s", digest)
	}

	if len(batchGetImageOutput.Images) > 2 {
		return dockerTypes.ImageInspect{}, fmt.Errorf("greater than 2 releases found for digest %s", digest)
	}

	return s.inspect(ctx, registryId, repository, batchGetImageOutput)
}

func (s Service) inspect(ctx context.Context, registryId, repository string, batchGetImageOutput *ecr.BatchGetImageOutput) (dockerTypes.ImageInspect, error) {
	var inspect dockerTypes.ImageInspect
	var distributionManifest DistributionManifest
	var downloadUrlResp *ecr.GetDownloadUrlForLayerOutput
	var resp *http.Response
	var body []byte

	manifestJson := []byte(*batchGetImageOutput.Images[0].ImageManifest)
	if err := json.Unmarshal(manifestJson, &distributionManifest); err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	configDigest := distributionManifest.Config.Digest

	urlInput := &ecr.GetDownloadUrlForLayerInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repository),
		LayerDigest:    aws.String(configDigest),
	}

	downloadUrlResp, err := s.Client.Ecr.GetDownloadUrlForLayer(ctx, urlInput)
	if err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	downloadUrl := downloadUrlResp.DownloadUrl

	resp, err = http.Get(*downloadUrl)
	if err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	err = json.Unmarshal(body, &inspect)
	if err != nil {
		return dockerTypes.ImageInspect{}, err
	}

	return inspect, nil
}

func (s Service) ImageUri(ctx context.Context, registryId, registryUrl, repository, tag string) (string, error) {
	input := &ecr.BatchGetImageInput{
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repository),
		ImageIds: []ecrTypes.ImageIdentifier{
			{
				ImageTag: aws.String(tag),
			},
		},
	}

	output, err := s.Client.Ecr.BatchGetImage(ctx, input)
	if err != nil {
		return "", err
	}

	digest := *output.Images[0].ImageId.ImageDigest
	imageUri := registryUrl + "/" + repository + "@" + digest
	return imageUri, nil
}
