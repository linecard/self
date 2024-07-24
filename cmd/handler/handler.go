package handler

import (
	"context"
	"fmt"
	"path"

	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var cfg config.Config
var api sdk.API

// Listen for events from the AWS Lambda runtime.
func Listen(tp *sdktrace.TracerProvider) {
	instrumented := otellambda.InstrumentHandler(Handler,
		otellambda.WithTracerProvider(tp),
		otellambda.WithFlusher(tp),
	)

	lambda.Start(instrumented)
}

// Handler function to process ECR image action events.
func Handler(ctx context.Context, event events.ECRImageActionEvent) error {
	BeforeEach(ctx, event)

	if util.ShaLike(event.Detail.ImageTag) {
		log.Warn().
			Str("repository", event.Detail.RepositoryName).
			Str("sha", cfg.Git.Sha).
			Msg("skipping")
		return nil
	}

	ctx, span := otel.Tracer("").Start(ctx, "handler")
	defer span.End()

	switch event.Detail.ActionType {
	case "PUSH":

		log.Info().
			Str("function", event.Detail.RepositoryName).
			Str("branch", cfg.Git.Branch).
			Msgf("deploying")

		span.SetAttributes(
			attribute.String("self.deploy.repository", event.Detail.RepositoryName),
			attribute.String("self.deploy.branch", event.Detail.ImageTag),
			attribute.String("self.deploy.function", path.Base(event.Detail.RepositoryName)),
		)

		release, err := api.Release.Find(ctx, event.Detail.RepositoryName, event.Detail.ImageTag)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to find release: %v", err)
		}

		deployment, err := api.Deployment.Deploy(ctx, release)
		if err != nil {
			return fmt.Errorf("failed to deploy release: %v", err)
		}

		if err := api.Subscription.Converge(ctx, deployment); err != nil {
			return fmt.Errorf("failed to converge subscriptions: %v", err)
		}

		if err := api.Httproxy.Converge(ctx, deployment); err != nil {
			return fmt.Errorf("failed to converge gateway httproxy: %v", err)
		}

	case "DELETE":
		log.Info().
			Str("repository", event.Detail.RepositoryName).
			Str("branch", event.Detail.ImageTag).
			Msg("destroying")

		span.SetAttributes(
			attribute.String("self.destroy.repository", event.Detail.RepositoryName),
			attribute.String("self.destroy.branch", event.Detail.ImageTag),
			attribute.String("self.destroy.function", path.Base(event.Detail.RepositoryName)),
		)

		deployment, err := api.Deployment.Find(ctx, util.DeSlasher(event.Detail.RepositoryName))
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

	default:
		err := fmt.Errorf("action type %s not supported", event.Detail.ActionType)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}
