package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/sdk"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var cfg config.Config
var api sdk.API

// Listen for events from the AWS Lambda runtime.
func Listen(ctx context.Context) {
	lambda.Start(Handler)
}

// Handler function to process ECR image action events.
func Handler(ctx context.Context, event events.ECRImageActionEvent) error {
	BeforeEach(ctx, event)

	if util.ShaLike(event.Detail.ImageTag) {
		log.Printf("skipping deployment for sha-like tag %s", event.Detail.ImageTag)
		return nil
	}

	switch event.Detail.ActionType {
	case "PUSH":
		log.Printf("deploying %s:%s", cfg.Function.Name, cfg.Git.Branch)

		release, err := api.Release.Find(ctx, cfg.Git.Branch)
		if err != nil {
			log.Fatalf("failed to find release: %v", err)
		}

		deployment, err := api.Deployment.Deploy(ctx, release, cfg.Git.Branch, cfg.Function.Name)
		if err != nil {
			log.Fatalf("failed to deploy release: %v", err)
		}

		if err := api.Subscription.Converge(ctx, deployment); err != nil {
			log.Fatalf("failed to converge subscriptions: %v", err)
		}

		if err := api.Httproxy.Converge(ctx, deployment, cfg.Git.Branch); err != nil {
			log.Fatalf("failed to converge gateway httproxy: %v", err)
		}

	case "DELETE":
		log.Printf("destroying %s:%s", cfg.Function.Name, cfg.Git.Branch)

		deployment, err := api.Deployment.Find(ctx, cfg.Git.Branch, cfg.Function.Name)
		if err != nil {
			log.Fatalf("failed to find deployment: %v", err)
		}

		subscriptions, err := api.Subscription.List(ctx, deployment)
		if err != nil {
			log.Fatalf("failed to list subscriptions: %v", err)
		}

		for _, subscription := range subscriptions {
			if err := api.Subscription.Disable(ctx, deployment, subscription); err != nil {
				log.Fatalf("failed to disable subscription: %v", err)
			}
		}

		if err := api.Httproxy.Unmount(ctx, deployment); err != nil {
			log.Fatalf("failed to umount gateway httproxy: %v", err)
		}

		if err = api.Deployment.Destroy(ctx, deployment); err != nil {
			log.Fatalf("failed to destroy deployment: %v", err)
		}

	default:
		msg := fmt.Sprintf("action type %s not supported", event.Detail.ActionType)
		return fmt.Errorf(msg)
	}

	return nil
}
