package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/release"
	"github.com/linecard/self/pkg/service/docker"
	"github.com/rs/zerolog/log"
)

type RuntimeService interface {
	Deploy(ctx context.Context, input docker.DeployInput) error
}

type Services struct {
	Runtime RuntimeService
}

type Convention struct {
	Config  config.Config
	Service Services
}

func FromServices(c config.Config, r RuntimeService) Convention {
	return Convention{
		Config: c,
		Service: Services{
			Runtime: r,
		},
	}
}

func (c Convention) Emulate(ctx context.Context, i release.Image) error {
	command := append(i.Config.Entrypoint, i.Config.Cmd...)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	riePath, err := ensureRieBinary(homeDir)
	if err != nil {
		return err
	}

	deploytime, err := c.Config.DeployTime(i.Config.Labels)
	if err != nil {
		return err
	}

	creds, err := c.Config.AssumeRoleWithPolicy(ctx, deploytime.Policy.Decoded)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to assume role with policy")
	}

	deployInput := docker.DeployInput{
		RiePath:         riePath,
		Region:          c.Config.Account.Region,
		ImageUri:        i.RepoTags[0],
		Function:        deploytime.Computed.Resource.Name,
		Command:         command,
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
	}

	if err := c.Service.Runtime.Deploy(ctx, deployInput); err != nil {
		return err
	}

	return nil
}

func ensureRieBinary(homeDir string) (string, error) {
	var rieUrl string

	switch runtime.GOARCH {
	case "amd64", "x86_64":
		rieUrl = "https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/download/v1.21/aws-lambda-rie-x86_64"
	case "arm64", "aarch64":
		rieUrl = "https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/download/v1.21/aws-lambda-rie-arm64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	dir := homeDir + "/.aws-lambda-rie"
	file := dir + "/aws-lambda-rie"
	var resp *http.Response
	var out *os.File

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	if _, err := os.Stat(file); err != nil {
		if resp, err = http.Get(rieUrl); err != nil {
			return "", err
		}

		defer resp.Body.Close()

		if out, err = os.Create(file); err != nil {
			return "", err
		}

		if _, err = io.Copy(out, resp.Body); err != nil {
			return "", err
		}

		if err = os.Chmod(file, 0755); err != nil {
			return "", err
		}
	}

	return file, nil
}
