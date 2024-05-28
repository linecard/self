# Concepts

Self is architected in a typical layered approach, striving to seperate out three primary categories from one another.
1. Convention - A layer of decisions based on GitOps.
1. Service - A layer of idempotent wrappers around services (AWS, Docker, etc).
1. Clients - A layer of out-of-the-box clients (AWS SDK v2 mostly).

The Convention layer is the most useful to explore, as the types and methods it exposes reprisent the abstractions surfaced to the end user.

## Release
A release is an image built by Self via the function's `Dockerfile`.

## Deployment
A deployment is an invokable release.

## Invoke
The most basic function execution.

## Bus
A bus is a pub/sub which can invoke a deployment. 

## HTTProxy
The httproxy is a reverse proxy url which can invoke a deployment.