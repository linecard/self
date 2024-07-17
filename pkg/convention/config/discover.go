package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/util"
)

type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

type ECRClient interface {
	DescribeRegistry(ctx context.Context, params *ecr.DescribeRegistryInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRegistryOutput, error)
}

func (c *Config) DiscoverCaller(ctx context.Context, client STSClient, awsConfig aws.Config) (err error) {
	var req *sts.GetCallerIdentityInput
	var res *sts.GetCallerIdentityOutput

	if res, err = client.GetCallerIdentity(ctx, req); err != nil {
		return err
	}

	c.Caller.Arn = *res.Arn
	c.Account.Id = *res.Account
	c.Account.Region = awsConfig.Region
	return nil
}

func (c *Config) DiscoverRegistry(ctx context.Context, ecrEnvar, regionEnvar string, ecrFallback ECRClient, regionFallback aws.Config) (err error) {
	var req *ecr.DescribeRegistryInput
	var res *ecr.DescribeRegistryOutput

	if region, exists := os.LookupEnv(regionEnvar); exists {
		c.Registry.Region = region
	} else {
		c.Registry.Region = regionFallback.Region
	}

	if id, exists := os.LookupEnv(ecrEnvar); exists {
		c.Registry.Id = id
	} else {
		res, err = ecrFallback.DescribeRegistry(ctx, req)
		if err != nil {
			return err
		}
		c.Registry.Id = *res.RegistryId
	}

	c.Registry.Url = c.Registry.Id + ".dkr.ecr." + c.Registry.Region + ".amazonaws.com"

	return nil
}

func (c *Config) DiscoverGateway(ctx context.Context, envar string) (err error) {
	if gwId, exists := os.LookupEnv(envar); exists {
		c.ApiGateway.Id = &gwId
	}
	return nil
}

func (c *Config) DiscoverVpc(ctx context.Context, sgEnvar, snEnvar string) (err error) {
	var count int

	if sgIds, sgExists := os.LookupEnv(sgEnvar); sgExists {
		splitIds := strings.Split(sgIds, ",")
		c.Vpc.SecurityGroupIds = splitIds
		count++
	}

	if snIds, snExists := os.LookupEnv(snEnvar); snExists {
		splitIds := strings.Split(snIds, ",")
		c.Vpc.SubnetIds = splitIds
		count++
	}

	if count != 0 && count != 2 {
		return fmt.Errorf("either both or none of AWS_SECURITY_GROUP_IDS and AWS_SUBNET_IDS must be set")
	}

	return nil
}

func (c *Config) DiscoverGit(ctx context.Context) (err error) {
	if git, err := gitlib.FromCwd(); err == nil {
		c.Git.Branch = git.Branch
		c.Git.Sha = git.Sha
		c.Git.Root = git.Root
		c.Git.Origin = git.Origin.String()
		c.Git.Dirty = git.Dirty
		return nil
	}
	return err
}

func (c *Config) DiscoverFunctions(ctx context.Context) (err error) {
	selfish := func(path string) bool {
		signature := []string{"policy.json.tmpl", "Dockerfile"}

		for _, item := range signature {
			fullPath := fmt.Sprintf("%s/%s", path, item)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				return false
			}
		}

		return true
	}

	filepath.Walk(c.Git.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && selfish(path) {

			name := filepath.Base(path)
			resourcePrefix := strings.Replace(strings.TrimPrefix(c.Git.Origin, "/"), ".git", "", 1)
			urlPrefix := strings.Replace(filepath.Base(c.Git.Origin), ".git", "", 1)
			c.Functions = append(c.Functions, ReleaseSchema{
				Path: path,
				Computed: Computed{
					Registry: ComputedRegistry{
						Url: c.Registry.Url,
					},
					Repository: ComputedRepository{
						Prefix: urlPrefix,
						Name:   name,
						Url:    fmt.Sprintf("%s/%s", c.Registry.Url, name),
					},
					Resource: ComputedResource{
						Prefix: resourcePrefix,
						Name:   resourcePrefix + "-" + util.DeSlasher(c.Git.Branch) + "-" + name,
					},
				},
				Schema: StringLabel{
					Description: "Label schema version string",
					Key:         LabelKeys.Schema,
					Content:     "1.1",
					Required:    true,
				},
				Name: StringLabel{
					Description: "Function name string",
					Key:         LabelKeys.Name,
					Content:     name,
					Required:    true,
				},
				Branch: StringLabel{
					Description: "Git branch string",
					Key:         LabelKeys.Branch,
					Content:     c.Git.Branch,
					Required:    true,
				},
				Sha: StringLabel{
					Description: "Git sha string",
					Key:         LabelKeys.Sha,
					Content:     c.Git.Sha,
					Required:    true,
				},
				Origin: StringLabel{
					Description: "Git origin string",
					Key:         LabelKeys.Origin,
					Content:     c.Git.Origin,
					Required:    true,
				},
				Role: EmbeddedFileLabel{
					Description: "Role template file",
					Key:         LabelKeys.Role,
					Path:        "embedded/roles/lambda.json.tmpl",
					Required:    true,
				},
				Policy: FileLabel{
					Description: "Policy template file",
					Key:         LabelKeys.Policy,
					Path:        filepath.Join(path, "policy.json.tmpl"),
					Required:    true,
				},
				Resources: FileLabel{
					Description: "Resources template file",
					Key:         LabelKeys.Resources,
					Path:        filepath.Join(path, "resources.json.tmpl"),
				},
				Bus: FolderLabel{
					Description: "Bus templates path",
					KeyPrefix:   LabelKeys.Bus,
					Path:        filepath.Join(path, "bus"),
				},
			})
		}

		return nil
	})

	return nil
}
