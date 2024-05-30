# Continuous Deployment

Self achieves continuous deployment by existing as a deployed release which listens to ECR `PUSH` and `DELETE` events. Once continuous deployment is enabled, the developer flow graph is reduced to it's minimal state, requiring zero developer interaction.

## Flow

```mermaid
graph LR
    develop([develop])
    push([push])

    develop --> push --> develop
```

## Deploy Self with Self

```bash
self init self continuous-deployment
cd continuous-deployment
self publish --ecr-login --ensure-repository
self deploy

self enable # this will enable the lambda to begin continuously deploying.
self disable # this will disable the lambda from continuously deploying.
```





