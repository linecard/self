package function

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	types "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
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

func (s Service) PutFunction(ctx context.Context, name string, roleArn string, imageUri string, arch types.Architecture, ephemeralStorage, memorySize, timeout int32, subnetIds []string, tags map[string]string) (*lambda.GetFunctionOutput, error) {
	var apiErr smithy.APIError

	getFunctionInput := &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	}

	createFunctionInput := &lambda.CreateFunctionInput{
		FunctionName:  aws.String(name),
		Role:          aws.String(roleArn),
		Architectures: []types.Architecture{arch},
		Code: &types.FunctionCode{
			ImageUri: aws.String(imageUri),
		},
		PackageType: types.PackageTypeImage,
		EphemeralStorage: &types.EphemeralStorage{
			Size: aws.Int32(ephemeralStorage),
		},
		MemorySize: aws.Int32(memorySize),
		Timeout:    aws.Int32(timeout),
		Tags:       tags,
	}

	// might be better allowing subnetIds to be nil from the bottom up?
	if len(subnetIds) > 0 {
		createFunctionInput.VpcConfig = &types.VpcConfig{
			SubnetIds: subnetIds,
		}
	}

	putFunctionConcurrencyInput := &lambda.PutFunctionConcurrencyInput{
		FunctionName:                 aws.String(name),
		ReservedConcurrentExecutions: aws.Int32(5),
	}

	getFunctionOutput, err := s.Client.Lambda.GetFunction(ctx, getFunctionInput)
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ResourceNotFoundException":
			_, err := s.Client.Lambda.CreateFunction(ctx, createFunctionInput, func(options *lambda.Options) {
				options.Retryer = retry.AddWithErrorCodes(options.Retryer, (*types.InvalidParameterValueException)(nil).ErrorCode())
				options.Retryer = retry.AddWithMaxAttempts(options.Retryer, 10)
			})

			if err != nil {
				return &lambda.GetFunctionOutput{}, err
			}

			_, err = s.Client.Lambda.PutFunctionConcurrency(ctx, putFunctionConcurrencyInput)
			if err != nil {
				return &lambda.GetFunctionOutput{}, err
			}

			return s.Client.Lambda.GetFunction(ctx, getFunctionInput)
		default:
			return &lambda.GetFunctionOutput{}, err
		}
	}

	updateLambdaConfigurationInput := lambda.UpdateFunctionConfigurationInput{
		FunctionName:     aws.String(name),
		Role:             aws.String(roleArn),
		ImageConfig:      createFunctionInput.ImageConfig,
		MemorySize:       createFunctionInput.MemorySize,
		EphemeralStorage: createFunctionInput.EphemeralStorage,
		Timeout:          createFunctionInput.Timeout,
		VpcConfig:        createFunctionInput.VpcConfig,
	}

	updateFunctionCodeInput := lambda.UpdateFunctionCodeInput{
		FunctionName:  aws.String(name),
		ImageUri:      aws.String(imageUri),
		Architectures: createFunctionInput.Architectures,
		Publish:       true,
	}

	_, err = s.Client.Lambda.UpdateFunctionConfiguration(ctx, &updateLambdaConfigurationInput)

	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	_, err = s.Client.Lambda.UpdateFunctionCode(ctx, &updateFunctionCodeInput, func(options *lambda.Options) {
		options.Retryer = retry.AddWithErrorCodes(options.Retryer, (*types.ResourceConflictException)(nil).ErrorCode())
		options.Retryer = retry.AddWithMaxAttempts(options.Retryer, 10)
	})

	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	_, err = s.Client.Lambda.PutFunctionConcurrency(ctx, putFunctionConcurrencyInput)
	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	tagResourceInput := lambda.TagResourceInput{
		Resource: aws.String(*getFunctionOutput.Configuration.FunctionArn),
		Tags:     tags,
	}

	_, err = s.Client.Lambda.TagResource(ctx, &tagResourceInput)

	if err != nil {
		return &lambda.GetFunctionOutput{}, err
	}

	return s.Client.Lambda.GetFunction(ctx, getFunctionInput)
}

func (s Service) DeleteFunction(ctx context.Context, name string) (*lambda.DeleteFunctionOutput, error) {
	deleteInput := lambda.DeleteFunctionInput{
		FunctionName: aws.String(name),
	}

	return s.Client.Lambda.DeleteFunction(ctx, &deleteInput)
}
