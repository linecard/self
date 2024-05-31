# Source & Dependencies

When you initialize a function within a local git repository with `self init` you are provided a minimal `Dockerfile`.

When you run `self publish` this `Dockerfile` is used to build the image that is pushed to ECR.

You can manipulate this `Dockerfile` as you see fit.

## References
- [Dockerfile](https://docs.docker.com/reference/dockerfile/)
- [OCI Images and Lambda](https://docs.aws.amazon.com/lambda/latest/dg/images-create.html)
