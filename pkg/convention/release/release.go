package release

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/smithy-go"
	"github.com/docker/docker/api/types"
	"github.com/golang-module/carbon/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type RegistryService interface {
	InspectByTag(ctx context.Context, registryId, repository, tag string) (types.ImageInspect, error)
	ImageUri(ctx context.Context, registryId, registryUrl, repository, tag string) (string, error)
	List(ctx context.Context, registryId, repository string) (ecr.DescribeImagesOutput, error)
	Delete(ctx context.Context, registryId, repository string, imageDigests []string) error
	Untag(ctx context.Context, registryId, repository, tag string) error
	PutRepository(ctx context.Context, repositoryName string) error
}

type BuildService interface {
	InspectByTag(ctx context.Context, registryUrl, repository, tag string) (types.ImageInspect, error)
	Build(ctx context.Context, path string, labels map[string]string, tags []string) error
	Push(ctx context.Context, tag string) error
}

type Image struct {
	types.ImageInspect
}

type Release struct {
	Image
	Uri string
}

type ReleaseSummary struct {
	Branch      string
	GitSha      string
	ImageDigest string
	Released    string
}

type Service struct {
	Registry RegistryService
	Build    BuildService
}

type Convention struct {
	Config  config.Config
	Service Service
}

func FromServices(c config.Config, r RegistryService, b BuildService) Convention {
	return Convention{
		Config: c,
		Service: Service{
			Registry: r,
			Build:    b,
		},
	}
}

func (c Convention) Find(ctx context.Context, tag string) (Release, error) {
	ctx, span := otel.Tracer("").Start(ctx, "release.Find")
	defer span.End()

	repository := c.Config.Repository.Prefix + "/" + c.Config.Function.Name
	inspect, err := c.Service.Registry.InspectByTag(ctx, c.Config.Registry.Id, repository, tag)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Release{}, err
	}

	uri, err := c.Service.Registry.ImageUri(ctx, c.Config.Registry.Id, c.Config.Registry.Url, repository, tag)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Release{}, err
	}

	return Release{Image{inspect}, uri}, nil
}

func (c Convention) List(ctx context.Context, function string) ([]ReleaseSummary, error) {
	var releases []ReleaseSummary
	var apiErr smithy.APIError
	repository := c.Config.Repository.Prefix + "/" + function

	list, err := c.Service.Registry.List(ctx, c.Config.Registry.Url, repository)
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "RepositoryNotFoundException":
			return []ReleaseSummary{}, nil
		default:
			return []ReleaseSummary{}, err
		}
	}

	for _, image := range list.ImageDetails {
		summary := ReleaseSummary{}
		summary.ImageDigest = string(*image.ImageDigest)
		summary.Released = image.ImagePushedAt.String()
		for _, tag := range image.ImageTags {
			if util.ShaLike(tag) {
				summary.GitSha = string(tag)
			} else {
				summary.Branch = tag
			}
		}
		releases = append(releases, summary)
	}

	return releases, nil
}

func (c Convention) GcPlan(ctx context.Context, functionName string) ([]ReleaseSummary, []string, error) {
	releases, err := c.List(ctx, functionName)
	if err != nil {
		return []ReleaseSummary{}, []string{}, err
	}

	var saveDigests []ReleaseSummary
	var deleteDigests []string

	now := carbon.CreateFromStdTime(time.Now())

	for _, release := range releases {
		released := carbon.Parse(release.Released)

		if release.Branch == "" && release.GitSha == "" {
			deleteDigests = append(deleteDigests, release.ImageDigest)
		} else if release.Branch == "" && released.Lt(now.SubWeeks(4)) { // parameterize this some day.
			deleteDigests = append(deleteDigests, release.ImageDigest)
		} else {
			saveDigests = append(saveDigests, release)
		}
	}

	return saveDigests, deleteDigests, nil
}

func (c Convention) GcApply(ctx context.Context, functionName string, digests []string) error {
	repository := c.Config.Repository.Prefix + "/" + functionName
	return c.Service.Registry.Delete(ctx, c.Config.Registry.Id, repository, digests)
}

func (c Convention) Publish(ctx context.Context, i Image) error {
	// This is a very fuzzy validation. Long run we wont need it. Nice to catch blatant problems for now.
	if len(i.RepoTags) != 2 {
		for _, tag := range i.RepoTags {
			fmt.Println(tag)
		}
		return fmt.Errorf("image must have exactly two tags, was given %d, do you have any identical builds?", len(i.RepoTags))
	}

	for _, tag := range i.RepoTags {
		if err := c.Service.Build.Push(ctx, tag); err != nil {
			return err
		}
	}

	return nil
}

func (c Convention) Build(ctx context.Context, functionPath, branch, sha string) (Image, error) {
	registryUrl := c.Config.Registry.Url
	repository := c.Config.Repository.Prefix + "/" + c.Config.Function.Name

	// coerce label sha to given sha
	labels, err := c.Config.Labels.Encode(&sha)
	if err != nil {
		return Image{}, err
	}

	tags := []string{
		fmt.Sprintf("%s/%s:%s", registryUrl, repository, branch),
		fmt.Sprintf("%s/%s:%s", registryUrl, repository, sha),
	}

	if err := c.Service.Build.Build(ctx, functionPath, labels, tags); err != nil {
		return Image{}, err
	}

	imageInspect, err := c.Service.Build.InspectByTag(ctx, registryUrl, repository, sha)
	if err != nil {
		return Image{}, err
	}

	return Image{imageInspect}, nil
}

func (c Convention) Untag(ctx context.Context, tag string) error {
	repository := c.Config.Repository.Prefix + "/" + c.Config.Function.Name
	return c.Service.Registry.Untag(ctx, c.Config.Registry.Id, repository, tag)
}

func (c Convention) EnsureRepository(ctx context.Context) error {
	repositoryName := c.Config.Repository.Prefix + "/" + c.Config.Function.Name
	return c.Service.Registry.PutRepository(ctx, repositoryName)
}
