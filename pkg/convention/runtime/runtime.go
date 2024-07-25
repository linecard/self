package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/release"
	"github.com/linecard/self/pkg/service/docker"
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

func (c Convention) Emulate(ctx context.Context, i release.Image, s *types.Credentials) error {
	command := append(i.Config.Entrypoint, i.Config.Cmd...)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	riePath, err := ensureRieBinary(homeDir)
	if err != nil {
		return err
	}

	_, computed, err := c.Config.Parse(i.Config.Labels)
	if err != nil {
		return err
	}

	deployInput := docker.DeployInput{
		RiePath:         riePath,
		Region:          c.Config.Account.Region,
		ImageUri:        i.RepoTags[0],
		Function:        computed.Resource.Name,
		Command:         command,
		AccessKeyId:     *s.AccessKeyId,
		SecretAccessKey: *s.SecretAccessKey,
		SessionToken:    *s.SessionToken,
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
		rieUrl = "https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/download/v1.18/aws-lambda-rie-x86_64"
	case "arm64", "aarch64":
		rieUrl = "https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/download/v1.18/aws-lambda-rie-arm64"
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
