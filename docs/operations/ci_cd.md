# CI & CD

When you implement both Continuous Integration and Continuous Deployment, the development flow collapses to...

## Flow

```mermaid
graph LR
    develop([develop])
    push(["git push"])

    develop --> push --> develop
```