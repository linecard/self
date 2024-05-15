package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/convention/release"
	"github.com/linecard/self/internal/labelgun"
	"github.com/linecard/self/internal/util"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	dockerTypes "github.com/docker/docker/api/types"
)

type FunctionService interface {
	Inspect(ctx context.Context, name string) (*lambda.GetFunctionOutput, error)
	List(ctx context.Context, prefix string) ([]lambda.GetFunctionOutput, error)
	PutPolicy(ctx context.Context, arn string, document string, tags map[string]string) (*iam.GetPolicyOutput, error)
	DeletePolicy(ctx context.Context, arn string) (*iam.DeletePolicyOutput, error)
	PutRole(ctx context.Context, name string, document string, tags map[string]string) (*iam.GetRoleOutput, error)
	DeleteRole(ctx context.Context, name string) (*iam.DeleteRoleOutput, error)
	AttachPolicyToRole(ctx context.Context, policyArn, roleName string) (*iam.AttachRolePolicyOutput, error)
	DetachPolicyFromRole(ctx context.Context, policyArn, roleName string) (*iam.DetachRolePolicyOutput, error)
	PutFunction(ctx context.Context, name string, roleArn string, imageUri string, arch types.Architecture, ephemeralStorage, memorySize, timeout int32, tags map[string]string) (*lambda.GetFunctionOutput, error)
	DeleteFunction(ctx context.Context, name string) (*lambda.DeleteFunctionOutput, error)
	GetRolePolicies(ctx context.Context, name string) (*iam.ListAttachedRolePoliciesOutput, error)
}

type RegistryService interface {
	InspectByDigest(ctx context.Context, registryId, repository, digest string) (dockerTypes.ImageInspect, error)
}

type Deployment struct {
	lambda.GetFunctionOutput
}

type Services struct {
	Function FunctionService
	Registry RegistryService
}

type Convention struct {
	Config  config.Config
	Service Services
}

func FromServices(c config.Config, f FunctionService, r RegistryService) Convention {
	return Convention{
		Config: c,
		Service: Services{
			Function: f,
			Registry: r,
		},
	}
}

func (c Convention) Find(ctx context.Context, namespace, functionName string) (Deployment, error) {
	resource := c.Config.ResourceName(namespace, functionName)
	lambda, err := c.Service.Function.Inspect(ctx, resource)
	if err != nil {
		return Deployment{}, err
	}

	return Deployment{*lambda}, nil
}

func (c Convention) List(ctx context.Context) ([]Deployment, error) {
	var deployments []Deployment
	lambdas, err := c.Service.Function.List(ctx, c.Config.Resource.Prefix)
	if err != nil {
		return []Deployment{}, err
	}

	for _, lambda := range lambdas {
		deployments = append(deployments, Deployment{lambda})
	}

	return deployments, nil
}

func (c Convention) ListNameSpace(ctx context.Context, namespace string) ([]Deployment, error) {
	var deployments []Deployment
	lambdas, err := c.Service.Function.List(ctx, c.Config.Resource.Prefix)
	if err != nil {
		return []Deployment{}, err
	}

	for _, lambda := range lambdas {
		if lambda.Tags["NameSpace"] == namespace {
			deployments = append(deployments, Deployment{lambda})
		}
	}

	return deployments, nil
}

func (c Convention) Deploy(ctx context.Context, release release.Release, namespace, functionName string) (Deployment, error) {
	resource := c.Config.ResourceName(namespace, functionName)

	roleTemplate, err := labelgun.DecodeLabel(c.Config.Label.Role, release.Config.Labels)
	if err != nil {
		return Deployment{}, err
	}

	roleDocument, err := c.Config.Template(roleTemplate)
	if err != nil {
		return Deployment{}, err
	}

	policyTemplate, err := labelgun.DecodeLabel(c.Config.Label.Policy, release.Config.Labels)
	if err != nil {
		return Deployment{}, err
	}

	policyDocument, err := c.Config.Template(policyTemplate)
	if err != nil {
		return Deployment{}, err
	}

	sha, err := labelgun.DecodeLabel(c.Config.Label.Sha, release.Config.Labels)
	if err != nil {
		return Deployment{}, err
	}

	resources := struct {
		EphemeralStorage int32 `json:"ephemeralStorage"`
		MemorySize       int32 `json:"memorySize"`
		Timeout          int32 `json:"timeout"`
	}{
		EphemeralStorage: 1024,
		MemorySize:       1024,
		Timeout:          120,
	}

	if labelgun.HasLabel(c.Config.Label.Resources, release.Config.Labels) {
		resourcesTemplate, err := labelgun.DecodeLabel(c.Config.Label.Resources, release.Config.Labels)
		if err != nil {
			return Deployment{}, err
		}

		resourcesDocument, err := c.Config.Template(resourcesTemplate)
		if err != nil {
			return Deployment{}, err
		}

		if err := json.Unmarshal([]byte(resourcesDocument), &resources); err != nil {
			return Deployment{}, err
		}
	}

	imageUri := release.Uri
	imageArch, err := toArch(release.Architecture)

	if err != nil {
		return Deployment{}, err
	}

	tags := map[string]string{"NameSpace": namespace, "Function": functionName, "Sha": sha}

	role, err := c.Service.Function.PutRole(ctx, resource, roleDocument, tags)
	if err != nil {
		return Deployment{}, err
	}

	policyArn := util.PolicyArnFromName(c.Config.Account.Id, resource)
	policy, err := c.Service.Function.PutPolicy(ctx, policyArn, policyDocument, tags)
	if err != nil {
		return Deployment{}, err
	}

	if _, err := c.Service.Function.AttachPolicyToRole(ctx, *policy.Policy.Arn, *role.Role.RoleName); err != nil {
		return Deployment{}, err
	}

	if _, err = c.Service.Function.PutFunction(
		ctx,
		resource,
		*role.Role.Arn,
		imageUri,
		imageArch,
		resources.EphemeralStorage,
		resources.MemorySize,
		resources.Timeout,
		tags,
	); err != nil {
		return Deployment{}, err
	}

	return c.Find(ctx, namespace, functionName)
}

func (c Convention) Destroy(ctx context.Context, d Deployment) error {
	roleName := util.RoleNameFromArn(*d.Configuration.Role)

	policies, err := c.Service.Function.GetRolePolicies(ctx, roleName)
	if err != nil {
		return err
	}

	for _, policy := range policies.AttachedPolicies {
		if _, err := c.Service.Function.DetachPolicyFromRole(ctx, *policy.PolicyArn, roleName); err != nil {
			return err
		}

		if _, err := c.Service.Function.DeletePolicy(ctx, *policy.PolicyArn); err != nil {
			return err
		}
	}

	if _, err = c.Service.Function.DeleteRole(ctx, roleName); err != nil {
		return err
	}

	if _, err = c.Service.Function.DeleteFunction(ctx, *d.Configuration.FunctionName); err != nil {
		return err
	}

	return nil
}

func (d Deployment) FetchRelease(ctx context.Context, r RegistryService, registryId string) (release.Release, error) {
	pathIndex := strings.Index(*d.Code.ImageUri, "/")
	imageTag := string(*d.Code.ImageUri)[pathIndex+1:]
	repository := strings.Split(imageTag, "@sha256:")[0]
	digest := *d.Configuration.CodeSha256

	fetched, err := r.InspectByDigest(ctx, registryId, repository, digest)
	if err != nil {
		return release.Release{}, err
	}

	return release.Release{Image: release.Image{ImageInspect: fetched}, Uri: *d.Code.ImageUri}, nil
}

func toArch(arch string) (types.Architecture, error) {
	switch arch {
	case "arm64":
		return types.Architecture("arm64"), nil
	case "amd64":
		return types.Architecture("x86_64"), nil
	case "x86_64":
		return types.Architecture("x86_64"), nil
	default:
		return "", fmt.Errorf("unsupported architecture %s", arch)
	}
}
