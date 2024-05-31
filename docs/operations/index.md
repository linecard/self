# Operations

Self can be used locally as a tool for publishing and deployment of releases. 

It can also be setup in such a manner that developers in your organization need not use the tool at all, driven completely by GitOps and CI/CD.

## CLI Behavior

Self uses it's surroundings to automatically populate large swaths of information typically required when using such a tool.

There are two scopes of operation for the Self cli:
1. Within a repository.
1. Within a function directory in that repository.

When you are operating `self` within a repository, but you are not in a function directory, you will only be presented with a subset of commands such as listing releases and deployments.

When you are operating `self` within a function directory, you will be presented with the full suite of commands, because self assumes that if you are _in_ a functions root directory, that _is_ the function you intend to operate against.

This can be further understood by running...

```bash
self config | jq .
```

... in each of these scenarios.

## CI Behavior

In a CI environment, `self` is used in the same way as it is locally. You will need to provide it credentials via the standard AWS credential chain.

## CD Behavior

For CD, `self` is published and deployed to AWS Lambda using `self`. It is aware of when it is running in a Lambda, and expects events from ECR. When evented, it calls `deploy` on `PUSH` events and `destroy` on `DELETE` events.