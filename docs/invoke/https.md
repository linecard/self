# HTTPs Invoke

By default, `self init` creates https based functions as it is the most user-friendly as well as powerful way to present your Lambda for consumption.

When your `resources.json.tmpl` contains...

```json
{
    "http": true,
    "public": true
}
```

Your Lambda will be mounted to the [discoverable API Gateway](/requirements?id=api-gateway-setup) in your account.

The convention for mounting is as follows...

```
https://<api-id>.execute-api.<region>.amazonaws.com/${repo}/${branch}/${function}
```

You can use a Custom Domain to replace `<api-id>.execute-api.<region>.amazonaws.com` with a domain of your choosing, this is outside of the scope of Self.

# IAM Auth

You can place IAM authentication in front of your route by setting the `public` key in `resources.json.tmpl` to `false`.

This is very useful for building internal tooling and services as you can place the URL on the public internet, but only those with the correct IAM role can access it.