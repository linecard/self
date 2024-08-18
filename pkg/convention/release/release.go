package release

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
	"github.com/docker/docker/api/types"
	"github.com/golang-module/carbon/v2"
)

type RegistryService interface {
	InspectByTag(ctx context.Context, registryId, repositoryName, tag string) (types.ImageInspect, error)
	ImageUri(ctx context.Context, registryId, registryUrl, repositoryName, tag string) (string, error)
	List(ctx context.Context, registryId, repositoryName string) (ecr.DescribeImagesOutput, error)
	Delete(ctx context.Context, registryId, repositoryName string, imageDigests []string) error
	Untag(ctx context.Context, registryId, repositoryName, tag string) error
	PutRepository(ctx context.Context, repositoryName string) error
}

type BuildService interface {
	InspectByTag(ctx context.Context, registryUrl, repository, tag string) (types.ImageInspect, error)
	Build(ctx context.Context, functionPath, contextPath string, labels map[string]string, tags []string) error
	Push(ctx context.Context, tag string) error
}

type EventService interface {
	Emit(ctx context.Context, accountId, busName, detailType string, detail any) error
}

type Image struct {
	types.ImageInspect
}

type Release struct {
	Image
	Uri             string
	AWSArchitecture []lambdatypes.Architecture
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
	Event    EventService
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

func (c Convention) Find(ctx context.Context, repositoryName, tag string) (Release, error) {
	ctx, span := otel.Tracer("").Start(ctx, "find")
	defer span.End()

	span.SetAttributes(
		attribute.String("registry-url", c.Config.Registry.Url),
		attribute.String("registry-id", c.Config.Registry.Id),
		attribute.String("repository-name", repositoryName),
		attribute.String("tag", tag),
	)

	inspect, err := c.Service.Registry.InspectByTag(ctx, c.Config.Registry.Id, repositoryName, tag)
	if err != nil {
		return Release{}, err
	}

	uri, err := c.Service.Registry.ImageUri(ctx, c.Config.Registry.Id, c.Config.Registry.Url, repositoryName, tag)
	if err != nil {
		return Release{}, err
	}

	var awsArch []lambdatypes.Architecture
	switch inspect.Architecture {
	case "arm64":
		awsArch = append(awsArch, "arm64")
	case "amd64":
		awsArch = append(awsArch, "x86_64")
	case "x86_64":
		awsArch = append(awsArch, "x86_64")
	default:
		return Release{}, fmt.Errorf("unsupported architecture %s", inspect.Architecture)
	}

	span.SetAttributes(
		attribute.String("image-id", inspect.ID),
		attribute.StringSlice("image-digest", inspect.RepoDigests),
		attribute.StringSlice("image-tags", inspect.RepoTags),
		attribute.String("image-uri", uri),
	)

	return Release{Image{inspect}, uri, awsArch}, nil
}

func (c Convention) List(ctx context.Context, repositoryName string) ([]ReleaseSummary, error) {
	var releases []ReleaseSummary
	var apiErr smithy.APIError

	list, err := c.Service.Registry.List(ctx, c.Config.Registry.Url, repositoryName)
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

func (c Convention) Build(ctx context.Context, path, context string) (Image, config.BuildTime, error) {
	ctx, span := otel.Tracer("").Start(ctx, "build")
	defer span.End()

	buildtime, err := c.Config.BuildTime(path)
	if err != nil {
		return Image{}, buildtime, err
	}

	tags := []string{ // make this a part of computed.
		fmt.Sprintf("%s:%s",
			buildtime.Computed.Repository.Url,
			buildtime.Branch.Decoded,
		),
		fmt.Sprintf("%s:%s",
			buildtime.Computed.Repository.Url,
			buildtime.Sha.Decoded,
		),
	}

	span.SetAttributes(
		attribute.String("build-path", path),
		attribute.String("build-context", context),
		attribute.String("branch", buildtime.Branch.Decoded),
		attribute.String("sha", buildtime.Sha.Decoded),
		attribute.String("origin", buildtime.Origin.Decoded),
		attribute.Bool("dirty", c.Config.Git.Dirty),
		attribute.StringSlice("tags", tags),
	)

	err = c.Service.Build.Build(
		ctx,
		path,
		context,
		buildtime.EncodedLabels(),
		tags,
	)

	if err != nil {
		return Image{}, buildtime, err
	}

	inspect, err := c.Service.Build.InspectByTag(
		ctx,
		buildtime.Computed.Registry.Url,
		buildtime.Computed.Repository.Name,
		buildtime.Sha.Decoded,
	)

	if err != nil {
		return Image{}, buildtime, err
	}

	return Image{inspect}, buildtime, nil
}

func (c Convention) Publish(ctx context.Context, i Image) error {
	ctx, span := otel.Tracer("").Start(ctx, "publish")
	defer span.End()

	span.SetAttributes(
		attribute.String("image-id", i.ID),
		attribute.StringSlice("image-digest", i.RepoDigests),
		attribute.StringSlice("image-tags", i.RepoTags),
	)

	// This is a very fuzzy validation. Catches issues with messy commits and mutable tagging ecr-side.
	if len(i.RepoTags) != 2 {
		for _, tag := range i.RepoTags {
			fmt.Println(tag)
		}
		return fmt.Errorf("image must have exactly two tags, was given %d, try deleting local images", len(i.RepoTags))
	}

	for _, tag := range i.RepoTags {
		if err := c.Service.Build.Push(ctx, tag); err != nil {
			return err
		}
	}

	return nil
}

func (c Convention) Untag(ctx context.Context, repositoryName, tag string) error {
	ctx, span := otel.Tracer("").Start(ctx, "untag")
	defer span.End()

	span.SetAttributes(
		attribute.String("registry-url", c.Config.Registry.Url),
		attribute.String("registry-id", c.Config.Registry.Id),
		attribute.String("repository-name", repositoryName),
		attribute.String("tag", tag),
	)

	return c.Service.Registry.Untag(ctx, c.Config.Registry.Id, repositoryName, tag)
}

func (c Convention) EnsureRepository(ctx context.Context, repositoryName string) error {
	return c.Service.Registry.PutRepository(ctx, repositoryName)
}

func (c Convention) GcPlan(ctx context.Context, repositoryName string) ([]ReleaseSummary, []string, error) {
	releases, err := c.List(ctx, repositoryName)
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

func (c Convention) GcApply(ctx context.Context, repositoryName string, digests []string) error {
	return c.Service.Registry.Delete(ctx, c.Config.Registry.Id, repositoryName, digests)
}
