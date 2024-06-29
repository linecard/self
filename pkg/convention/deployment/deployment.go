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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/rs/zerolog/log"
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
	DeleteFunction(ctx context.Context, name string) (*lambda.DeleteFunctionOutput, error)
	GetRolePolicies(ctx context.Context, name string) (*iam.ListAttachedRolePoliciesOutput, error)
	PutFunction(ctx context.Context, put *lambda.CreateFunctionInput, concurreny int32) (*lambda.GetFunctionOutput, error)
	PatchFunction(ctx context.Context, patch *lambda.UpdateFunctionConfigurationInput) (*lambda.GetFunctionConfigurationOutput, error)
	EnsureEniGcRole(ctx context.Context) (*iam.GetRoleOutput, error)
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
	var err error

	ctx, span := otel.Tracer("").Start(ctx, "deployment.Deploy")
	defer span.End()

	resource := c.Config.ResourceName(namespace, functionName)

	// if vpc configuration is partial, return an error.
	if (c.Config.Vpc.SubnetIds == nil) != (c.Config.Vpc.SecurityGroupIds == nil) {
		err := fmt.Errorf("VPC configuration requires both subnet and security group IDs to be set, or neither")
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	// pull labels from release and base64 decode.
	labels, err := c.Config.Labels.Decode(release.Config.Labels)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	// template decoded labels.
	for k, v := range labels {
		templatedValue, err := c.Config.Template(v)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return Deployment{}, err
		}

		labels[k] = templatedValue
	}

	// set default values for pertinent resources.json.tmpl settings.
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

	// grab image uri and architecture from release for deployment parameters.
	imageUri := release.Uri
	imageArch, err := toArch(release.Architecture)
	if err != nil {
		return Deployment{}, err
	}

	// grab some label data for deployment tagging.
	sha := labels[c.Config.Labels.Sha.Key]
	roleDocument := labels[c.Config.Labels.Role.Key]
	policyDocument := labels[c.Config.Labels.Policy.Key]
	tags := map[string]string{"NameSpace": namespace, "Function": functionName, "Sha": sha}

	// create role
	role, err := c.Service.Function.PutRole(ctx, resource, roleDocument, tags)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	// create policy
	policyArn := util.PolicyArnFromName(c.Config.Account.Id, resource)
	policy, err := c.Service.Function.PutPolicy(ctx, policyArn, policyDocument, tags)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	// mix um together
	if _, err := c.Service.Function.AttachPolicyToRole(ctx, *policy.Policy.Arn, *role.Role.RoleName); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	// create function parameters
	input := &lambda.CreateFunctionInput{
		FunctionName:  aws.String(resource),
		Role:          role.Role.Arn,
		Tags:          tags,
		Architectures: []types.Architecture{imageArch},
		PackageType:   types.PackageTypeImage,
		Timeout:       &resources.Timeout,
		MemorySize:    &resources.MemorySize,
		VpcConfig: &types.VpcConfig{
			SecurityGroupIds: c.Config.Vpc.SecurityGroupIds,
			SubnetIds:        c.Config.Vpc.SubnetIds,
		},
		EphemeralStorage: &types.EphemeralStorage{
			Size: &resources.EphemeralStorage,
		},
		Code: &types.FunctionCode{
			ImageUri: aws.String(imageUri),
		},
		Publish: true,
	}

	// create function inside of vpc land if configured to do so.
	if input.VpcConfig.SubnetIds != nil && input.VpcConfig.SecurityGroupIds != nil {
		log.Info().Msg("VPC configuration detected, ensuring ENI garbage collection role")

		eniRole, err := c.Service.Function.EnsureEniGcRole(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return Deployment{}, err
		}

		// The function must be created with a seperate (and persistent) role, as this role is used during garbage collection by ec2.
		// If you just launch with the desired role, that role will be deleted on destroy before garbage collection can clear the eni.
		// So all functions launched by self into vpcs use the AWSLambdaVPCAccessExecutionRole. It uses the managed policy of the same name.
		input.Role = eniRole.Role.Arn
		if _, err = c.Service.Function.PutFunction(ctx, input, 5); err != nil {
			span.SetStatus(codes.Error, err.Error())
			return Deployment{}, err
		}

		// After creating the function with this ENI garbage collection role, we can go ahead and attach the role we actually want.
		_, err = c.Service.Function.PatchFunction(ctx, &lambda.UpdateFunctionConfigurationInput{
			FunctionName: aws.String(resource),
			Role:         role.Role.Arn,
		})

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return Deployment{}, err
		}

		return c.Find(ctx, namespace, functionName)
	}

	// create function outside of vpc land.
	if _, err = c.Service.Function.PutFunction(ctx, input, 5); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return Deployment{}, err
	}

	return c.Find(ctx, namespace, functionName)
}

func (c Convention) Destroy(ctx context.Context, d Deployment) error {
	ctx, span := otel.Tracer("").Start(ctx, "deployment.Destroy")
	defer span.End()

	roleName := util.RoleNameFromArn(*d.Configuration.Role)

	if roleName != "AWSLambdaVPCAccessExecutionRole" {
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
	}

	if _, err := c.Service.Function.DeleteFunction(ctx, *d.Configuration.FunctionName); err != nil {
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
