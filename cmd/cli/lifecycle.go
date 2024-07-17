package cli

import (
	"context"
	"os"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jedib0t/go-pretty/table"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func BeforeAll(ctx context.Context) {
	var err error

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.With().Caller()

	tablec = table.NewWriter()
	tablec.SetOutputMirror(os.Stdout)

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

	cfg = config.Config{}
	if err := cfg.FromCwd(ctx, awsConfig, ecrc, stsc); err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration from cwd")
	}

	os.Setenv("DEFAULT_BRANCH", cfg.Git.Branch)
	os.Setenv("DEFAULT_SHA", cfg.Git.Sha)
	os.Setenv("DEFAULT_ENSURE_REPOSITORY", "false")

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SDK")
	}
}
