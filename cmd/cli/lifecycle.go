package cli

import (
	"context"
	"log"
	"os"

	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/umwelt"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/jedib0t/go-pretty/table"
)

func BeforeAll(ctx context.Context) {
	var err error

	tablec = table.NewWriter()
	tablec.SetOutputMirror(os.Stdout)

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf(err.Error())
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf(err.Error())
	}

	stsc = sts.NewFromConfig(awsConfig)
	ecrc := ecr.NewFromConfig(awsConfig)
	gwc := apigatewayv2.NewFromConfig(awsConfig)

	git, err := gitlib.FromCwd()
	if err != nil {
		log.Fatalf(err.Error())
	}

	here, err := umwelt.FromCwd(ctx, cwd, git, awsConfig, ecrc, gwc, stsc)
	if err != nil {
		log.Fatalf(err.Error())
	}

	cfg = config.FromHere(here)

	os.Setenv("DEFAULT_RELEASE_BRANCH", cfg.Git.Branch)
	os.Setenv("DEFAULT_RELEASE_SHA", cfg.Git.Sha)
	os.Setenv("DEFAULT_DEPLOYMENT_TAG", cfg.Git.Branch)
	os.Setenv("DEFAULT_DEPLOYMENT_NAMESPACE", cfg.Git.Branch)

	if api, err = sdk.Init(ctx, awsConfig, cfg); err != nil {
		log.Fatalf(err.Error())
	}
}
