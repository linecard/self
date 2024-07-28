package config

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
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

func templateString(document string, data TemplateData) (string, error) {
	tmpl, err := template.New("document").Parse(string(document))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
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

func (c Config) Scaffold(templateName, functionName string) error {
	scaffoldPath := "embedded/scaffold"
	templatePath := filepath.Join(scaffoldPath, templateName)

	if _, err := embedded.ReadDir(templatePath); os.IsNotExist(err) {
		templates, err := embedded.ReadDir(scaffoldPath)
		if err != nil {
			return err
		}

		var templateNames []string
		for _, template := range templates {
			templateNames = append(templateNames, template.Name())
		}

		return fmt.Errorf("scaffold %s does not exist. valid options: %s", templateName, strings.Join(templateNames, ", "))
	}

	return fs.WalkDir(embedded, templatePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path with respect to templatePath
		relPath, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}
		targetFilePath := filepath.Join(functionName, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetFilePath, os.ModePerm)
		}

		content, err := fs.ReadFile(embedded, path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return err
		}

		outputFile, err := os.Create(targetFilePath)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		err = tmpl.Execute(outputFile, c)
		if err != nil {
			return err
		}

		return nil
	})
}
