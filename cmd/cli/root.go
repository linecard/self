package cli

import (
	"context"
	"os"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/linecard/self/cmd/cli/router"
	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Invoke(ctx context.Context) {
	var err error
	var cfg config.Config
	var api sdk.API
	var stsc *sts.Client

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Caller().Logger()

	retryLogger := util.RetryLogger{
		Log: &log.Logger,
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithLogger(&retryLogger),
		awsconfig.WithClientLogMode(aws.LogRetries))

	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AWS configuration")
	}

	stsc = sts.NewFromConfig(awsConfig)
	ecrc := ecr.NewFromConfig(awsConfig)

	var root router.Root
	arg.MustParse(&root)

	if root.GlobalOpts.Branch != "" {
		os.Setenv(config.EnvGitBranch, root.GlobalOpts.Branch)
	}

	if root.GlobalOpts.Sha != "" {
		os.Setenv(config.EnvGitSha, root.GlobalOpts.Sha)
	}

	if root.GlobalOpts.EcrId != "" {
		os.Setenv(config.EnvEcrId, root.GlobalOpts.EcrId)
	}

	if root.GlobalOpts.EcrRegion != "" {
		os.Setenv(config.EnvEcrRegion, root.GlobalOpts.EcrRegion)
	}

	if root.GlobalOpts.SubnetIds != "" {
		os.Setenv(config.EnvSnIds, root.GlobalOpts.SubnetIds)
	}

	if root.GlobalOpts.SecurityGroupIds != "" {
		os.Setenv(config.EnvSgIds, root.GlobalOpts.SecurityGroupIds)
	}

	if root.GlobalOpts.OwnerPrefixResources {
		os.Setenv(
			config.EnvOwnerPrefixResources,
			strconv.FormatBool(root.GlobalOpts.OwnerPrefixResources),
		)
	}

	if root.GlobalOpts.OwnerPrefixRoutes {
		os.Setenv(
			config.EnvOwnerPrefixRoutes,
			strconv.FormatBool(root.GlobalOpts.OwnerPrefixRoutes),
		)
	}

	if cfg, err = config.Stateful(ctx, awsConfig, stsc, ecrc); err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration from cwd")
	}

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SDK")
	}

	root.Route(ctx, api)
}
