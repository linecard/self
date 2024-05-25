# Self

Self provides developers the means to confidently build, verify, and deploy their code to AWS Lambda.

Self follows the 12-factor manifesto. Via convention and the services `self` is built upon, the developer is only responsible for those factors she is truly wanting to define: the deliverable artifact and runtime environment config.

# Infrastructure Setup

In order to truly feel the lift provided by `self` we are going to want to do a little bit of infrastructure work to prepare.

## Git Repo
`self` is gitops oriented. It refuses to work in a directory that is not:
1. A git repository.
1. with remote origin.

## API Gateway

While `self` doesn't require the usage of API Gateway inherently, we are going to start with it as it eliminates the need for lambda specific knowledge. It is also the most advisable approach to building lambdas as it abstracts the serverless interfaces almost entirely, allowing any developer to productively begin contributing without the truly unecessary overhead of learning lambdas "nuance".

Setup:
1. Login to your AWS account.
1. Goto the API Gateway service.
1. Click "Create API" => "HTTP API".
1. Create with all default settings.
1. Tag the api with `SelfManaged: true`.

If you have multiple `SelfManaged` APIs in your account, you can inform `self` which API gateway to target with the `AWS_API_GATEWAY_ID` environment variables.

## ECR Repository

It is entirely possible for `self` to automatically ensure ECR repositories are created for the deliverable OCI image artifact. 

It intentionally does not to encourage a moment of pause for design consideration.

Setup:
1. Login to your AWS account.
1. Goto the ECR service.
1. Create an ECR repository
1. Name it thusly: `{org/user}/{repo}/{function}`, extracted from `github.com/{org/user}/{repo}` and `{function}` being the name of the function you are creating.

If you use the ECR singleton pattern, where multiple accounts use a single ECR repository in another account, you can inform `self` of this with the `AWS_ECR_REGION` and `AWS_ECR_REGISTRY_ID` environment variables.

# Umwelt

Let's explore a core concept of `self` really quickly to make sense of _how_ it goes about doing what it does.

When you execute `self` in a directory it first has a look around using the `umwelt` module to configure the `sdk` module:
1. Derive that you are in a git repo, with a remote origin.
1. Looks for functions within the repository.
1. Looks to see if the current directory you are in has a function.
1. Looks to see if it can find an ECR repository.
1. Looks to see if it can find an API gateway (not required, but just use one).

Note that the tool is using _everything_ in it's ability to know about the world surrounding it. It is rabidly avoiding any configuration that cannot be determined by using sensory input about it's filesystem context as well as AWS caller identity.

This knowledge `self` induces about the world around it flattens the operational interface greatly as we will see.

# Show me already...

Let's assume you have your git repo setup with remote origin `https://github.com/user/myFunctions`

## Deploy an example
```
self init python explore
cd explore
self publish -l
self deploy
```

Goto the API gateway console and deduce the pathing and URL to which this was deployed. Of course this will be clarified to painful extent, but for now it is an exercise for the reader.

## Dive a little deeper
```
# run this in the root of the repo, then run it when in a function directory
self -h
```

```
# run this in the root of the repo, then run it when in a function directory
self config | jq .
```

# Note

This `README` is not yet covering the _actual_ usage of self for daily driving. This is the manual run through, with operational interfaces that aim to be improved. There are many features, like the accurate local emulation of lambdas, that have been left out. 

Spoiler: `self init self deployer` implements `self` within a lambda, deployed by `self` which continuously deploys/destroys lambdas in tight conjunction with git branches. `self` as a CLI tool is intended purely for operations, and a subset is to be exposed for local development flow (local lambda deployment).