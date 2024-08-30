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
	"go.opentelemetry.io/otel"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Invoke() {
	var err error
	var cfg config.Config
	var api sdk.API
	var stsc *sts.Client

	ctx := context.Background()
	ctx, span := otel.Tracer("").Start(ctx, "continuous-integration")
	defer span.End()

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

	configEnv(root)

	if cfg, err = config.Stateful(ctx, awsConfig, stsc, ecrc); err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration from cwd")
	}

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SDK")
	}

	if err := root.Route(ctx, api); err != nil {
		log.Fatal().Err(err).Strs("argv", os.Args).Msgf("failed command")
	}
}

// Take options given to the CLI and export them to their respective environment variables.
func configEnv(root router.Root) {
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

	if root.GlobalOpts.SelfBusName != "" {
		os.Setenv(config.EnvBusName, root.GlobalOpts.SelfBusName)
	}
}
