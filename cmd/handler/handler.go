package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func eventToCarrierMapper(eventJSON []byte) propagation.TextMapCarrier {
	var event config.Event

	log.Info().Msg("mapping event to carrier")

	err := json.Unmarshal(eventJSON, &event)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal event")
		return propagation.MapCarrier{} // Return empty carrier if event parsing fails
	}

	// Populate the carrier with traceparent and tracestate
	carrier := propagation.MapCarrier{}
	if event.Detail.Traceparent != "" {
		carrier.Set("traceparent", event.Detail.Traceparent)
	}
	if event.Detail.Tracestate != "" {
		carrier.Set("tracestate", event.Detail.Tracestate)
	}

	return carrier
}

// Listen for events from the AWS Lambda runtime.
func Listen(tp *sdktrace.TracerProvider) {
	instrumented := otellambda.InstrumentHandler(Handler,
		otellambda.WithTracerProvider(tp),
		otellambda.WithFlusher(tp),
		otellambda.WithEventToCarrier(eventToCarrierMapper),
	)

	lambda.Start(instrumented)
}

func Handler(ctx context.Context, event config.Event) (err error) {
	var cfg config.Config
	var api sdk.API

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetName("continuous-deployment")
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		span.SetStatus(codes.Code(codes.Error), "failed to load AWS configuration")
		return err
	}

	stsc := sts.NewFromConfig(awsConfig)
	ecrc := ecr.NewFromConfig(awsConfig)

	if cfg, err = config.Stateless(ctx, awsConfig, stsc, ecrc, event); err != nil {
		span.SetStatus(codes.Code(codes.Error), "failed to load configuration from event")
		return
	}

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		span.SetStatus(codes.Code(codes.Error), "failed to initialize SDK")
		return
	}

	for _, account := range event.Detail.ExceptAccounts {
		if account == cfg.Account.Id {
			span.SetStatus(codes.Code(codes.Ok), "skipping account as instructed by event")
			return nil
		}
	}

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
	deployment, err := api.Deployment.Find(ctx, event.ResourceName)
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
