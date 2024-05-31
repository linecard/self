# Requirements

## Install

Add a [release of self](https://github.com/linecard/self/releases) to your `PATH`.

## Git Ops

The Self cli will refuse to operate outside of a git repository configured with a remote origin. It will also refuse to publish a release if the branch is dirty.

## Credentials

Self must be able to find AWS credentials via the [credential chain](https://docs.aws.amazon.com/sdkref/latest/guide/standardized-credentials.html) providing access to the following services.

* [AWS ECR](https://aws.amazon.com/ecr/)
* [AWS Lambda](https://aws.amazon.com/lambda/)
* [AWS EventBridge](https://aws.amazon.com/eventbridge/)
* [AWS API Gateway](https://docs.aws.amazon.com/apigateway/)

## Dependencies

You must have...
* [Docker](https://www.docker.com/).
* A dedicated AWS API Gateway for Self to manage.

> **Note:** The API Gateway is _technically_ not a requirement. But usage of Self without it falls into the category of "Advanced". If you are already a Lambda expert, you will come to find that Self fully supports purely invoke and eventbridge driven patterns without http based invocation. However, operating at this lower level does not provide benefit over a well documented REST api with an [AWS_LWA_PASS_THROUGH_PATH](https://github.com/awslabs/aws-lambda-web-adapter) for eventbridge support.

### API Gateway Setup

1. Login to AWS Console.
1. Navigate to the API Gateway Service.
1. Create an HTTP API Gateway with all default settings.
1. Add a the tag `SelfDiscovery: <chosen name>` to the gateway.



## Additional Config

### Multiple API Gateways

Self will discover your API gateway **iff** there is only one gateway tagged with `SelfDiscovery: <chosen name>`. If there are two or more API Gateways tagged, you will need to specify which you are targeting with the `AWS_API_GATEWAY_ID` environment variable.

Self will not interact with an API Gateway that is not tagged.

### Cross-Account ECR

Organizations that use multiple AWS accounts often use a singleton ECR repository for all accounts. Self supports this via the `AWS_ECR_REGISTRY_ID` and `AWS_ECR_REGISTRY_REGION` environment variables.
