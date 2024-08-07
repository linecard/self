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

func (c *Config) DiscoverRegistry(ctx context.Context, ecrFallback ECRClient, regionFallback aws.Config) (err error) {
	var req *ecr.DescribeRegistryInput
	var res *ecr.DescribeRegistryOutput

	if region, exists := os.LookupEnv(EnvEcrRegion); exists {
		c.Registry.Region = region
	} else {
		c.Registry.Region = regionFallback.Region
	}

	if id, exists := os.LookupEnv(EnvEcrId); exists {
		c.Registry.Id = id
	} else {
		res, err = ecrFallback.DescribeRegistry(ctx, req)
		if err != nil {
			return err
		}
		c.Registry.Id = *res.RegistryId
	}

	c.Registry.Url = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", c.Registry.Id, c.Registry.Region)

	return nil
}

func (c *Config) DiscoverGateway(ctx context.Context) (err error) {
	if gwId, exists := os.LookupEnv(EnvGwId); exists {
		c.ApiGateway.Id = &gwId
	}
	return nil
}

func (c *Config) DiscoverVpc(ctx context.Context) (err error) {
	var count int

	if sgIds, sgExists := os.LookupEnv(EnvSgIds); sgExists {
		splitIds := strings.Split(sgIds, ",")
		c.Vpc.SecurityGroupIds = splitIds
		count++
	}

	if snIds, snExists := os.LookupEnv(EnvSnIds); snExists {
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
	if c.Git, err = gitlib.FromCwd(); err != nil {
		return err
	}

	if value, exists := os.LookupEnv(EnvGitBranch); exists {
		c.Git.Branch = value
	}

	if value, exists := os.LookupEnv(EnvGitSha); exists {
		c.Git.Sha = value
	}

	return err
}

func (c *Config) DiscoverSelfish(ctx context.Context) (err error) {
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

		selfRepoEmbedded := "self/pkg/convention/config/embedded/scaffold/"
		if info.IsDir() && selfish(path) && !strings.Contains(path, selfRepoEmbedded) {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			c.Selfish = append(c.Selfish, Selfish{
				Path: abs,
				Name: filepath.Base(abs),
			})
		}

		return nil
	})

	return nil
}
