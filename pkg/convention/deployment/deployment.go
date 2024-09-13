package deployment

import (
	"context"
	"strings"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/release"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

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

func (c Convention) Find(ctx context.Context, deploymentName string) (Deployment, error) {
	lambda, err := c.Service.Function.Inspect(ctx, deploymentName)
	if err != nil {
		return Deployment{}, err
	}

	return Deployment{*lambda}, nil
}

func (c Convention) List(ctx context.Context, deploymentPrefix string) ([]Deployment, error) {
	var deployments []Deployment
	lambdas, err := c.Service.Function.List(ctx, deploymentPrefix)
	if err != nil {
		return []Deployment{}, err
	}

	for _, lambda := range lambdas {
		deployments = append(deployments, Deployment{lambda})
	}

	return deployments, nil
}

func (c Convention) Deploy(ctx context.Context, r release.Release) (Deployment, error) {
	ctx, span := otel.Tracer("").Start(ctx, "deploy")
	defer span.End()

	var err error

	deploytime, err := c.Config.DeployTime(r.Config.Labels)
	if err != nil {
		return Deployment{}, err
	}

	span.SetAttributes(
		attribute.String("deployment-name", deploytime.Computed.Resource.Name),
		attribute.String("deployment-role", deploytime.Computed.Resource.Role.Arn),
		attribute.String("deployment-policy", deploytime.Computed.Resource.Policy.Arn),
	)

	role, err := c.Service.Function.PutRole(
		ctx,
		deploytime.Computed.Resource.Name,
		deploytime.Role.Decoded,
		deploytime.Computed.Resource.Tags,
	)

	if err != nil {
		return Deployment{}, err
	}

	policy, err := c.Service.Function.PutPolicy(
		ctx,
		deploytime.Computed.Resource.Policy.Arn,
		deploytime.Policy.Decoded,
		deploytime.Computed.Resource.Tags,
	)

	if err != nil {
		return Deployment{}, err
	}

	_, err = c.Service.Function.AttachPolicyToRole(
		ctx,
		*policy.Policy.Arn,
		*role.Role.RoleName,
	)

	if err != nil {
		return Deployment{}, err
	}

	// create function parameters
	input := &lambda.CreateFunctionInput{
		FunctionName:  aws.String(deploytime.Computed.Resource.Name),
		Role:          role.Role.Arn,
		Tags:          deploytime.Computed.Resource.Tags,
		Architectures: r.AWSArchitecture,
		PackageType:   types.PackageTypeImage,
		Timeout:       &deploytime.Computed.Resources.Timeout,
		MemorySize:    &deploytime.Computed.Resources.MemorySize,
		EphemeralStorage: &types.EphemeralStorage{
			Size: &deploytime.Computed.Resources.EphemeralStorage,
		},
		VpcConfig: &types.VpcConfig{
			SecurityGroupIds: []string{},
			SubnetIds:        []string{},
		},
		Code: &types.FunctionCode{
			ImageUri: aws.String(r.Uri),
		},
		Publish: true,
	}

	// Has VPC Config
	if c.Config.Vpc.SecurityGroupIds != nil && c.Config.Vpc.SubnetIds != nil {
		input.VpcConfig = &types.VpcConfig{
			SecurityGroupIds: c.Config.Vpc.SecurityGroupIds,
			SubnetIds:        c.Config.Vpc.SubnetIds,
		}

		log.Info().Msg("VPC configuration detected, ensuring ENI garbage collection role")

		eniRole, err := c.Service.Function.EnsureEniGcRole(ctx)
		if err != nil {
			return Deployment{}, err
		}

		// The function must be created with a seperate (and persistent) role, as this role is used during garbage collection by ec2.
		// If you just launch with the desired role, that role will be deleted on destroy before garbage collection can clear the eni.
		// So all functions launched by self into vpcs use the AWSLambdaVPCAccessExecutionRole. It uses the managed policy of the same name.
		input.Role = eniRole.Role.Arn
		if _, err = c.Service.Function.PutFunction(ctx, input, 5); err != nil {
			return Deployment{}, err
		}

		// After creating the function with this ENI garbage collection role, we can go ahead and attach the role we actually want.
		_, err = c.Service.Function.PatchFunction(ctx, &lambda.UpdateFunctionConfigurationInput{
			FunctionName: aws.String(deploytime.Computed.Resource.Name),
			Role:         role.Role.Arn,
		})

		if err != nil {
			return Deployment{}, err
		}

		return c.Find(ctx, deploytime.Computed.Resource.Name)
	}

	// Does not have VPC config
	if _, err = c.Service.Function.PutFunction(ctx, input, 5); err != nil {
		return Deployment{}, err
	}

	return c.Find(ctx, deploytime.Computed.Resource.Name)
}

func (c Convention) Destroy(ctx context.Context, d Deployment) error {
	roleName := util.RoleNameFromArn(*d.Configuration.Role)

	if roleName != "AWSLambdaVPCAccessExecutionRole" {
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
	}

	if _, err := c.Service.Function.DeleteFunction(ctx, *d.Configuration.FunctionName); err != nil {
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
