package sigv4

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

type Service struct {
	creds  aws.Credentials
	region string
}

func FromClients(creds aws.Credentials, region string) Service {
	return Service{
		creds:  creds,
		region: region,
	}
}

func (s Service) SignRequest(ctx context.Context, request *http.Request) (err error) {
	var bytes []byte

	if bytes, err = io.ReadAll(request.Body); err != nil {
		return
	}

	signer := v4.NewSigner()
	bodyHash := sha256.Sum256(bytes)
	encodedPayload := hex.EncodeToString(bodyHash[:])

	err = signer.SignHTTP(
		ctx,
		s.creds,
		request,
		encodedPayload,
		"execute-api",
		s.region,
		time.Now(),
	)

	return
}

func (s Service) MockIAMAuthRequestCtx(ctx context.Context, callerArn string, req *http.Request) (err error) {
	var bytes []byte
	context := events.APIGatewayV2HTTPRequestContext{
		Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
			IAM: &events.APIGatewayV2HTTPRequestContextAuthorizerIAMDescription{
				UserARN: callerArn,
			},
		},
	}

	if bytes, err = json.Marshal(context); err != nil {
		return
	}

	req.Header.Set("x-amzn-request-context", string(bytes))

	return
}
