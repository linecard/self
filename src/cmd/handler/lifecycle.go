package handler

import (
	"context"
	"log"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/internal/umwelt"
	"github.com/linecard/self/sdk"

	"github.com/aws/aws-lambda-go/events"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func BeforeEach(ctx context.Context, event events.ECRImageActionEvent) {
	var err error

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	stsc := sts.NewFromConfig(awsConfig)
	ecrc := ecr.NewFromConfig(awsConfig)
	gwc := apigatewayv2.NewFromConfig(awsConfig)

	here, err := umwelt.FromEvent(ctx, event, awsConfig, ecrc, gwc, stsc)
	if err != nil {
		log.Fatalf(err.Error())
	}

	cfg = config.FromHere(here)

	api, err = sdk.Init(ctx, awsConfig, cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
}
