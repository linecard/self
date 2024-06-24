package cli

import (
	"context"
	"os"

	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/umwelt"
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

	tablec = table.NewWriter()
	tablec.SetOutputMirror(os.Stdout)

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to determine working directory")
	}

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

	git, err := gitlib.FromCwd()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to introspect git, are you in a git repository? does it have a remote origin?")
	}

	here, err := umwelt.FromCwd(ctx, cwd, git, awsConfig, ecrc, stsc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to introspect surrounding environment")
	}

	cfg = config.FromHere(here)

	os.Setenv("DEFAULT_RELEASE_BRANCH", cfg.Git.Branch)
	os.Setenv("DEFAULT_RELEASE_SHA", cfg.Git.Sha)
	os.Setenv("DEFAULT_DEPLOYMENT_TAG", cfg.Git.Branch)
	os.Setenv("DEFAULT_DEPLOYMENT_NAMESPACE", cfg.Git.Branch)
	os.Setenv("DEFAULT_ENSURE_REPOSITORY", "false")

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SDK")
	}
}
