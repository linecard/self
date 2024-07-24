package function

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	types "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
)

const (
	shortRetry = 3
	stdRetry   = 3
	longRetry  = 3
)

func (s Service) List(ctx context.Context, prefix string) ([]lambda.GetFunctionOutput, error) {
	var functions []lambda.GetFunctionOutput

	listFunctionsInput := &lambda.ListFunctionsInput{}
	listFunctionsOutput, err := s.Client.Lambda.ListFunctions(ctx, listFunctionsInput)
	if err != nil {
		return nil, err
	}

	for _, function := range listFunctionsOutput.Functions {
		if strings.HasPrefix(*function.FunctionName, prefix) {
			getFunctionInput := &lambda.GetFunctionInput{
				FunctionName: function.FunctionName,
			}

			getFunctionOutput, err := s.Client.Lambda.GetFunction(ctx, getFunctionInput)
			if err != nil {
				return nil, err
			}

			functions = append(functions, *getFunctionOutput)
		}
	}

	return functions, nil
}

func (s Service) PutFunction(ctx context.Context, put *lambda.CreateFunctionInput, concurreny int32) (*lambda.GetFunctionOutput, error) {
	var apiErr smithy.APIError
	update := false

	_, err := s.Client.Lambda.CreateFunction(ctx, put, func(options *lambda.Options) {
		options.Retryer = retry.AddWithErrorCodes(options.Retryer,
			(*types.InvalidParameterValueException)(nil).ErrorCode(),
		)
		options.Retryer = retry.AddWithMaxAttempts(options.Retryer, longRetry)
	})

	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ResourceConflictException":
			update = true
		default:
			return &lambda.GetFunctionOutput{}, err
		}
	}

	if update {
		patchConfig := &lambda.UpdateFunctionConfigurationInput{
			FunctionName: put.FunctionName,
			Role:         put.Role,
			MemorySize:   put.MemorySize,
			Timeout:      put.Timeout,
			VpcConfig:    put.VpcConfig,
		}

		patchCode := &lambda.UpdateFunctionCodeInput{
			FunctionName:  put.FunctionName,
			ImageUri:      put.Code.ImageUri,
			Architectures: put.Architectures,
			Publish:       true,
		}

		_, err = s.Client.Lambda.UpdateFunctionConfiguration(ctx, patchConfig, func(options *lambda.Options) {
			options.Retryer = retry.AddWithMaxAttempts(options.Retryer, longRetry)
			options.Retryer = retry.AddWithErrorCodes(options.Retryer,
				(*types.InvalidParameterValueException)(nil).ErrorCode(),
				(*types.ResourceConflictException)(nil).ErrorCode(),
			)
		})
		if err != nil {
			return &lambda.GetFunctionOutput{}, err
		}

		_, err = s.Client.Lambda.UpdateFunctionCode(ctx, patchCode, func(options *lambda.Options) {
			options.Retryer = retry.AddWithMaxAttempts(options.Retryer, shortRetry)
			options.Retryer = retry.AddWithErrorCodes(options.Retryer,
				(*types.ResourceConflictException)(nil).ErrorCode(),
			)
		})
		if err != nil {
			return &lambda.GetFunctionOutput{}, err
		}
	}

	_, err = s.Client.Lambda.PutFunctionConcurrency(ctx, &lambda.PutFunctionConcurrencyInput{
		FunctionName:                 put.FunctionName,
		ReservedConcurrentExecutions: aws.Int32(concurreny),
	})
	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	function, err := s.Client.Lambda.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: put.FunctionName,
	})
	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	tagResourceInput := lambda.TagResourceInput{
		Resource: aws.String(*function.Configuration.FunctionArn),
		Tags:     put.Tags,
	}
	_, err = s.Client.Lambda.TagResource(ctx, &tagResourceInput)
	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	return s.Client.Lambda.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: put.FunctionName,
	})
}

func (s Service) DeleteFunction(ctx context.Context, name string) (*lambda.DeleteFunctionOutput, error) {
	deleteInput := lambda.DeleteFunctionInput{
		FunctionName: aws.String(name),
	}

	log.Info().Msgf("Function %s being deleted", name)
	return s.Client.Lambda.DeleteFunction(ctx, &deleteInput)
}

func (s Service) PatchFunction(ctx context.Context, patch *lambda.UpdateFunctionConfigurationInput) (*lambda.GetFunctionConfigurationOutput, error) {
	_, err := s.Client.Lambda.UpdateFunctionConfiguration(ctx, patch, func(options *lambda.Options) {
		options.Retryer = retry.AddWithMaxAttempts(options.Retryer, stdRetry)
		options.Retryer = retry.AddWithErrorCodes(options.Retryer,
			(*types.InvalidParameterValueException)(nil).ErrorCode(),
			(*types.ResourceConflictException)(nil).ErrorCode())
	})
	if err != nil {
		return nil, err
	}

	return s.Client.Lambda.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: patch.FunctionName,
	})
}

func (s Service) EnsureEniGcRole(ctx context.Context) (*iam.GetRoleOutput, error) {
	policyArn := "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
	trust := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "TrustLambda",
				"Effect": "Allow",
				"Principal": {
					"Service": "lambda.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	role, err := s.PutRole(ctx, "AWSLambdaVPCAccessExecutionRole", trust, map[string]string{})
	if err != nil {
		return &iam.GetRoleOutput{}, err
	}

	_, err = s.AttachPolicyToRole(ctx, policyArn, *role.Role.RoleName)
	if err != nil {
		return &iam.GetRoleOutput{}, err
	}

	return role, nil
}
