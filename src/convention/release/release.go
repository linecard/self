package release

import (
	"context"
	"fmt"
	"time"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/internal/labelgun"
	"github.com/linecard/self/internal/util"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/golang-module/carbon/v2"
)

type RegistryService interface {
	InspectByTag(ctx context.Context, registryId, repository, tag string) (types.ImageInspect, error)
	ImageUri(ctx context.Context, registryId, registryUrl, repository, tag string) (string, error)
	List(ctx context.Context, registryId, repository string) (ecr.DescribeImagesOutput, error)
	Delete(ctx context.Context, registryId, repository string, imageDigests []string) error
	Untag(ctx context.Context, registryId, repository, tag string) error
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
	repository := c.Config.Repository.Prefix + "/" + c.Config.Function.Name
	inspect, err := c.Service.Registry.InspectByTag(ctx, c.Config.Registry.Id, repository, tag)

	if err != nil {
		return Release{}, err
	}

	uri, err := c.Service.Registry.ImageUri(ctx, c.Config.Registry.Id, c.Config.Registry.Url, repository, tag)
	if err != nil {
		return Release{}, err
	}

	return Release{Image{inspect}, uri}, nil
}

func (c Convention) List(ctx context.Context, function string) ([]ReleaseSummary, error) {
	var releases []ReleaseSummary
	repository := c.Config.Repository.Prefix + "/" + function

	list, err := c.Service.Registry.List(ctx, c.Config.Registry.Url, repository)
	if err != nil {
		return []ReleaseSummary{}, err
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
	busRoot := functionPath + "/bus"
	registryUrl := c.Config.Registry.Url
	repository := c.Config.Repository.Prefix + "/" + c.Config.Function.Name

	labels := make(map[string]string)

	lambdaRole, err := c.Config.ReadStatic("static/roles/lambda.json.tmpl")
	if err != nil {
		return Image{}, err
	}

	encodedRole, err := labelgun.EncodeString(lambdaRole, true)
	if err != nil {
		return Image{}, err
	}
	labels[c.Config.Label.Role] = encodedRole

	encodedPolicy, err := labelgun.EncodeFile(functionPath + "/policy.json.tmpl")
	if err != nil {
		return Image{}, err
	}
	labels[c.Config.Label.Policy] = encodedPolicy

	encodedSha, err := labelgun.EncodeString(sha, false)
	if err != nil {
		return Image{}, err
	}
	labels[c.Config.Label.Sha] = encodedSha

	if util.PathExists(functionPath + "/resources.json.tmpl") {
		encodedResourceConfig, err := labelgun.EncodeFile(functionPath + "/resources.json.tmpl")
		if err != nil {
			return Image{}, err
		}
		labels[c.Config.Label.Resources] = encodedResourceConfig
	}

	var busEncodings map[string]string
	if !util.PathExists(busRoot) {
		busEncodings = make(map[string]string)
	} else {
		busEncodings, err = labelgun.EncodePath(c.Config.Label.Bus, busRoot)
		if err != nil {
			return Image{}, err
		}
	}

	for busLabel, encodedExpression := range busEncodings {
		labels[busLabel] = encodedExpression
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
