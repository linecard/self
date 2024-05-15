package function

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

func (s Service) Inspect(ctx context.Context, name string) (*lambda.GetFunctionOutput, error) {
	getFunctionInput := &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	}

	return s.Client.Lambda.GetFunction(ctx, getFunctionInput)
}
