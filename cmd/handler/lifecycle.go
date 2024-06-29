package handler

import (
	"context"

	"github.com/linecard/self/internal/umwelt"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/aws/aws-lambda-go/events"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/rs/zerolog/log"
)

func BeforeEach(ctx context.Context, event events.ECRImageActionEvent) {
	var err error

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AWS configuration")
	}

	stsc := sts.NewFromConfig(awsConfig)
	ecrc := ecr.NewFromConfig(awsConfig)

	here, err := umwelt.FromEvent(ctx, event, awsConfig, ecrc, stsc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to introspect surrounding environment")
	}

	cfg = config.FromHere(here)

	api, err = sdk.Init(ctx, awsConfig, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize SDK")
	}
}
