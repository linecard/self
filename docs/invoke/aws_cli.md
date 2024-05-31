# AWS Cli Invoke

The AWS CLI can be used to invoke a Lambda function.

```bash
aws lambda invoke \
--function-name ${repository}-${branch}-${function} \
--cli-binary-format raw-in-base64-out \
--payload '{ "event": "payload" }' \
response

cat response
```