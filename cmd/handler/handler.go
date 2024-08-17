package handler

import (
	"context"
	"fmt"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Listen for events from the AWS Lambda runtime.
func Listen(tp *sdktrace.TracerProvider) {
	instrumented := otellambda.InstrumentHandler(Handler,
		otellambda.WithTracerProvider(tp),
		otellambda.WithFlusher(tp),
	)

	lambda.Start(instrumented)
}

func Handler(ctx context.Context, event config.Event) (err error) {
	var cfg config.Config
	var api sdk.API

	// attempt to associate trace with originating publish/untag emitted events.
	if event.Detail.Traceparent != "" {
		carrier := propagation.MapCarrier{
			"traceparent": event.Detail.Traceparent,
			"tracestate":  event.Detail.Tracestate,
		}

		propagator := propagation.TraceContext{}
		ctx = propagator.Extract(ctx, carrier)
	}

	ctx, span := otel.Tracer("").Start(ctx, "handler")
	defer span.End()

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AWS configuration")
	}

	stsc := sts.NewFromConfig(awsConfig)
	ecrc := ecr.NewFromConfig(awsConfig)

	if cfg, err = config.Stateless(ctx, awsConfig, stsc, ecrc, event); err != nil {
		msg := "failed to load configuration from event"
		span.SetStatus(codes.Code(codes.Error), msg)
		log.Fatal().Err(err).Msg(msg)
	}

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		msg := "failed to initialize SDK"
		span.SetStatus(codes.Code(codes.Error), msg)
		log.Fatal().Err(err).Msg(msg)
	}

	log.Info().Msgf("received event: %s", event.DetailType)
	log.Info().Msgf("repository: %s", event.Detail.RepositoryName)
	log.Info().Msgf("branch: %s", event.Detail.Branch)

	switch event.DetailType {
	case "Deploy":
		return Deploy(ctx, api, event.Detail)
	case "Destroy":
		return Destroy(ctx, api, event.Detail)
	default:
		return fmt.Errorf("unknown event type: %s", event.DetailType)
	}
}

func Deploy(ctx context.Context, api sdk.API, detail config.EventDetail) (err error) {
	ctx, span := otel.Tracer("").Start(ctx, "Deploy")
	defer span.End()

	release, err := api.Release.Find(ctx, detail.RepositoryName, detail.Branch)
	if err != nil {
		return fmt.Errorf("failed to find release: %v", err)
	}

	deployment, err := api.Deployment.Deploy(ctx, release)
	if err != nil {
		return fmt.Errorf("failed to deploy release: %v", err)
	}

	err = api.Subscription.Converge(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to converge subscriptions: %v", err)
	}

	if err := api.Httproxy.Converge(ctx, deployment); err != nil {
		return fmt.Errorf("failed to converge gateway httproxy: %v", err)
	}

	return nil
}

func Destroy(ctx context.Context, api sdk.API, event config.EventDetail) error {
	ctx, span := otel.Tracer("").Start(ctx, "Destroy")
	defer span.End()

	deployment, err := api.Deployment.Find(ctx, util.DeSlasher(event.RepositoryName))
	if err != nil {
		return fmt.Errorf("failed to find deployment: %v", err)
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %v", err)
	}

	for _, subscription := range subscriptions {
		if err := api.Subscription.Disable(ctx, deployment, subscription); err != nil {
			return fmt.Errorf("failed to disable subscription: %v", err)
		}
	}

	if err := api.Httproxy.Unmount(ctx, deployment); err != nil {
		return fmt.Errorf("failed to unmount gateway httproxy: %v", err)
	}

	if err = api.Deployment.Destroy(ctx, deployment); err != nil {
		return fmt.Errorf("failed to destroy deployment: %v", err)
	}

	return nil
}
