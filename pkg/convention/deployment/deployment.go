package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/release"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

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
	PutFunction(ctx context.Context, name string, roleArn string, imageUri string, arch types.Architecture, ephemeralStorage, memorySize, timeout int32, subnetIds, securityGroupIds []string, tags map[string]string) (*lambda.GetFunctionOutput, error)
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
	ctx, span := otel.Tracer("").Start(ctx, "deployment.Find")
	defer span.End()

	resource := c.Config.ResourceName(namespace, functionName)
	lambda, err := c.Service.Function.Inspect(ctx, resource)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
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
	ctx, span := otel.Tracer("").Start(ctx, "deployment.Deploy")
	defer span.End()

	resource := c.Config.ResourceName(namespace, functionName)

	labels, err := c.Config.Labels.Decode(release.Config.Labels)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	for k, v := range labels {
		templatedValue, err := c.Config.Template(v)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return Deployment{}, err
		}

		labels[k] = templatedValue
	}

	// Opting for parsing/default of label values to be left until last moment.
	// So long as these cases exist in single locations, it's better than hoisting it up to config in terms of clutter.
	resources := struct {
		EphemeralStorage int32 `json:"ephemeralStorage"`
		MemorySize       int32 `json:"memorySize"`
		Timeout          int32 `json:"timeout"`
	}{
		EphemeralStorage: 512,
		MemorySize:       128,
		Timeout:          3,
	}

	if _, exists := labels[c.Config.Labels.Resources.Key]; exists {
		if err := json.Unmarshal([]byte(labels[c.Config.Labels.Resources.Key]), &resources); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return Deployment{}, err
		}
	}

	imageUri := release.Uri
	imageArch, err := toArch(release.Architecture)
	if err != nil {
		return Deployment{}, err
	}

	sha := labels[c.Config.Labels.Sha.Key]
	roleDocument := labels[c.Config.Labels.Role.Key]
	policyDocument := labels[c.Config.Labels.Policy.Key]
	tags := map[string]string{"NameSpace": namespace, "Function": functionName, "Sha": sha}

	role, err := c.Service.Function.PutRole(ctx, resource, roleDocument, tags)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	policyArn := util.PolicyArnFromName(c.Config.Account.Id, resource)
	policy, err := c.Service.Function.PutPolicy(ctx, policyArn, policyDocument, tags)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	if _, err := c.Service.Function.AttachPolicyToRole(ctx, *policy.Policy.Arn, *role.Role.RoleName); err != nil {
		span.SetStatus(codes.Error, err.Error())
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
		c.Config.Vpc.SubnetIds,
		c.Config.Vpc.SecurityGroupIds,
		tags,
	); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	return c.Find(ctx, namespace, functionName)
}

func (c Convention) Destroy(ctx context.Context, d Deployment) error {
	ctx, span := otel.Tracer("").Start(ctx, "httproxy.Destroy")
	defer span.End()

	roleName := util.RoleNameFromArn(*d.Configuration.Role)

	policies, err := c.Service.Function.GetRolePolicies(ctx, roleName)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, policy := range policies.AttachedPolicies {
		if _, err := c.Service.Function.DetachPolicyFromRole(ctx, *policy.PolicyArn, roleName); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		if _, err := c.Service.Function.DeletePolicy(ctx, *policy.PolicyArn); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if _, err = c.Service.Function.DeleteRole(ctx, roleName); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if _, err = c.Service.Function.DeleteFunction(ctx, *d.Configuration.FunctionName); err != nil {
		span.SetStatus(codes.Error, err.Error())
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
