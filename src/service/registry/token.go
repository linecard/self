package registry

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

func (s Service) Token(ctx context.Context, registryId string) (string, error) {
	input := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []string{registryId},
	}

	output, err := s.Client.Ecr.GetAuthorizationToken(ctx, input)
	if err != nil {
		return "", err
	}

	encodedToken := *output.AuthorizationData[0].AuthorizationToken

	data, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(string(data), ":", 2)
	token := parts[1]

	return token, nil
}
