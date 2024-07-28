package cli

import (
	"context"
	"os"

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

	if err := cfg.FromCwd(ctx, awsConfig, ecrc, stsc); err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration from cwd")
	}

	// Override branch if provided
	if root.GitOpts.Branch != "" {
		cfg.Git.Branch = root.GitOpts.Branch
	}

	// Override sha if provided
	if root.GitOpts.Sha != "" {
		cfg.Git.Sha = root.GitOpts.Sha
	}

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SDK")
	}

	root.Route(ctx, api)
}
