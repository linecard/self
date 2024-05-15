package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types"
)

type Service struct {
	Root   string
	Binary string
}

type DeployInput struct {
	RiePath         string
	Region          string
	ImageUri        string
	Function        string
	Command         []string
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
}

func FromPath(ctx context.Context) (Service, error) {
	binary, err := exec.LookPath("docker")
	if err != nil {
		return Service{}, err
	}

	return Service{Binary: binary}, nil
}

func (s Service) Login(ctx context.Context, registryUrl, username, password string) error {
	cmd := exec.CommandContext(ctx, s.Binary, "login", "--username", username, "--password", password, registryUrl)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (s Service) InspectByTag(ctx context.Context, registryUrl, repository, tag string) (types.ImageInspect, error) {
	image := registryUrl + "/" + repository + ":" + tag
	cmd := exec.CommandContext(ctx, s.Binary, "image", "inspect", image)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return types.ImageInspect{}, err
	}

	var inspectData []types.ImageInspect
	if err := json.Unmarshal(output, &inspectData); err != nil {
		return types.ImageInspect{}, err
	}

	if len(inspectData) == 0 {
		return types.ImageInspect{}, fmt.Errorf("no image found for tag %s", tag)
	}

	if len(inspectData) > 1 {
		return types.ImageInspect{}, fmt.Errorf("multiple images found for tag %s", tag)
	}

	return inspectData[0], nil
}

func (s Service) Build(ctx context.Context, path string, labels map[string]string, tags []string) error {
	envs := []string{
		"DOCKER_BUILDKIT=1",
	}

	args := []string{
		"build",
		"-f", path + "/Dockerfile",
	}

	for _, tag := range tags {
		args = append(args, "-t", tag)
	}

	for key, value := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, path)

	cmd := exec.CommandContext(ctx, s.Binary, args...)
	cmd.Env = append(os.Environ(), envs...)
	cmd.Stderr = os.Stderr

	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func (s Service) Push(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, s.Binary, "push", tag)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (s Service) Deploy(ctx context.Context, i DeployInput) error {
	argv := []string{
		"run",
		"--rm",
		"-v", filepath.Dir(i.RiePath) + ":/.aws-lambda-rie",
		"-p", "9000:8080",
		"--env", "AWS_DEFAULT_REGION=" + i.Region,
		"--env", "AWS_ACCESS_KEY_ID=" + i.AccessKeyId,
		"--env", "AWS_SECRET_ACCESS_KEY=" + i.SecretAccessKey,
		"--env", "AWS_SESSION_TOKEN=" + i.SessionToken,
		"--env", "AWS_LAMBDA_FUNCTION_NAME=" + i.Function,
		"--entrypoint", "/.aws-lambda-rie/" + filepath.Base(i.RiePath),
		i.ImageUri,
	}

	argv = append(argv, i.Command...)

	cmd := exec.CommandContext(ctx, s.Binary, argv...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
