package config

import (
	"context"
	"html/template"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsc "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/rs/zerolog/log"

	"github.com/linecard/self/internal/util"
)

func (c Config) Template(document string) (string, error) {
	tmpl, err := template.New("document").Parse(string(document))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, c.TemplateData); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (c Config) AssumeRoleWithPolicy(ctx context.Context, policy string) (*types.Credentials, error) {
	var awsConf aws.Config
	var fallback aws.Credentials
	var arn string
	var err error

	if awsConf, err = awsc.LoadDefaultConfig(ctx); err != nil {
		return nil, err
	}

	if fallback, err = awsConf.Credentials.Retrieve(ctx); err != nil {
		return nil, err
	}

	if arn, err = util.RoleArnFromAssumeRoleArn(c.Caller.Arn); err == nil {
		var output *sts.AssumeRoleOutput

		stsc := sts.NewFromConfig(awsConf)
		output, err = stsc.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(arn),
			RoleSessionName: aws.String(os.Getenv("USER") + "-masquerade"),
			Policy:          &policy,
		})

		if err == nil {
			return output.Credentials, nil
		}
	}

	log.Warn().Err(err).Msg("falling back to local credential context")
	return &types.Credentials{
		AccessKeyId:     &fallback.AccessKeyID,
		SecretAccessKey: &fallback.SecretAccessKey,
		SessionToken:    &fallback.SessionToken,
	}, nil
}
